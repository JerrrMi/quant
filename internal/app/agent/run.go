package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain/auth"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/executor"
	"github.com/JerrrMi/quant/internal/infra/agentstate"
	"github.com/JerrrMi/quant/internal/infra/binance"
	"github.com/JerrrMi/quant/internal/infra/ws"
	"github.com/JerrrMi/quant/internal/lifecycle"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// Deps binds runtime effects for Agent (local store only—no SaaS control-plane DB URL).
type Deps struct {
	Logger *slog.Logger
	DB     *gorm.DB
}

// Run connects to SaaS over WebSocket, authenticates, and executes Binance Futures commands until ctx is cancelled.
func Run(ctx context.Context, cfg config.AgentConfig, deps Deps) error {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}

	apiKey := strings.TrimSpace(os.Getenv(cfg.Binance.APIKeyEnv))
	secret := strings.TrimSpace(os.Getenv(cfg.Binance.APISecretEnv))
	if cfg.Binance.PassphraseEnv != "" {
		_ = strings.TrimSpace(os.Getenv(cfg.Binance.PassphraseEnv))
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Exchange.Name), "binance") && (apiKey == "" || secret == "") {
		return fmt.Errorf("agent: set %s and %s in the environment before starting", cfg.Binance.APIKeyEnv, cfg.Binance.APISecretEnv)
	}

	bc := binance.NewUSDMMClient(binance.RESTBaseForConfig(cfg.Binance.UseTestnet), apiKey, secret, nil)
	var venue executor.Venue
	if strings.EqualFold(strings.TrimSpace(cfg.Exchange.Name), "binance") {
		venue = &executor.BinanceVenue{C: bc}
	} else {
		return fmt.Errorf("agent: unsupported exchange.name %q (only binance wired)", cfg.Exchange.Name)
	}

	if err := bc.SyncServerTime(ctx); err != nil {
		log.Warn("binance server time sync failed", "err", err)
	}

	dedup := agentstate.NewDedupStore(deps.DB)
	maxN := parseMaxNotional(cfg.Risk.MaxNotionalQuotePerOrder, log)
	execsvc := executor.NewService(venue, dedup, maxN, cfg.Risk.MaxOpenOrders)

	var saasSeq atomic.Int64
	policy := lifecycle.AgentReconnectPolicy(
		cfg.Reconnect.InitialBackoffSecs,
		cfg.Reconnect.MaxBackoffSecs,
		cfg.Reconnect.JitterRatio,
		cfg.Reconnect.MaxAttempts,
	)
	backoff := lifecycle.NewExpBackoff(policy)

	hooks := lifecycle.ReconnectHooks{
		OnDisconnected: func(reason error, disconnectedAt time.Time, cumulativeAttempts int) {
			log.Warn("agent transport disconnected",
				"err", reason,
				"disconnected_at", disconnectedAt,
				"cumulative_disconnect_events", cumulativeAttempts)
		},
		BeforeReconnect: func(nextBackoff time.Duration, sessionIndex int) {
			log.Info("agent reconnect scheduled",
				"sleep", nextBackoff.String(),
				"session_index", sessionIndex)
		},
	}

	return lifecycle.RunReconnectLoop(ctx, log, hooks, backoff, func(sessCtx context.Context) error {
		return runSession(sessCtx, cfg, log, execsvc, &saasSeq, func(ctx context.Context) error {
			backoff.Reset()
			if hooks.AfterAuthSuccess != nil {
				return hooks.AfterAuthSuccess(ctx)
			}
			return nil
		})
	})
}

func runSession(ctx context.Context, cfg config.AgentConfig, log *slog.Logger, execsvc *executor.Service, saasSeq *atomic.Int64, onAuthed func(context.Context) error) error {
	d := websocket.DefaultDialer
	if cfg.Connection.DialTimeoutSecs > 0 {
		d.HandshakeTimeout = time.Duration(cfg.Connection.DialTimeoutSecs) * time.Second
	}
	if cfg.Connection.TLSInsecureSkipVerify && strings.HasPrefix(strings.ToLower(cfg.Connection.SaasWSURL), "wss") {
		if d.TLSClientConfig == nil {
			d.TLSClientConfig = &tls.Config{}
		}
		d.TLSClientConfig.InsecureSkipVerify = true
	}

	dialCtx := ctx
	if cfg.Connection.DialTimeoutSecs > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, time.Duration(cfg.Connection.DialTimeoutSecs)*time.Second)
		defer cancel()
	}

	conn, _, err := d.DialContext(dialCtx, cfg.Connection.SaasWSURL, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer conn.Close()

	transport := &ws.WebsocketTransport{Conn: conn}
	peer := ws.NewPeer(ws.RoleAgent, transport, nil)

	var outbound ws.SeqAllocator
	var mu sync.Mutex
	send := func(ctx context.Context, typ ws.MsgType, seq int64, ack *int64, payload any) error {
		mu.Lock()
		defer mu.Unlock()
		return peer.Send(ctx, typ, seq, ack, payload)
	}

	am := auth.AuthMessage{
		ProtocolVersion: "ws-protocol-v1",
		ClientID:        cfg.Identity.ClientID,
		AgentID:         cfg.Identity.AgentID,
		Nonce:           uuid.New().String(),
		RequestedScopes: []auth.AuthScope{auth.ScopeExecutionWrite, auth.ScopeExecutionRead},
		LastSeenSaasSeq: saasSeq.Load(),
	}
	seq := outbound.Next()
	if err := send(ctx, ws.MsgAuth, seq, nil, am); err != nil {
		return fmt.Errorf("auth send: %w", err)
	}

	authEnv, err := peer.RecvValidated(ctx)
	if err != nil {
		return fmt.Errorf("auth recv: %w", err)
	}
	if authEnv.Type != ws.MsgAuth {
		return fmt.Errorf("expected auth response, got %s", authEnv.Type)
	}
	var ar auth.AuthResult
	if err := json.Unmarshal(authEnv.Payload, &ar); err != nil {
		return fmt.Errorf("auth decode: %w", err)
	}
	recordSaasSeq(saasSeq, authEnv.Seq)
	if !ar.OK {
		return fmt.Errorf("auth rejected code=%s msg=%s", ar.ErrorCode, ar.Message)
	}
	if onAuthed != nil {
		if err := onAuthed(ctx); err != nil {
			return fmt.Errorf("post-auth hook: %w", err)
		}
	}
	log.Info("agent authenticated with SaaS", "session_id", ar.SessionID, "reconnect_backoff_reset", true)

	hb := time.Duration(cfg.Connection.HeartbeatIntervalSecs) * time.Second
	if hb <= 0 {
		hb = ws.DefaultHeartbeatInterval
	}

	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	go func() {
		err := ws.RunHeartbeatTicker(hbCtx, hb, func(ts int64) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s := outbound.Next()
			return send(hbCtx, ws.MsgHeartbeat, s, nil, ws.HeartbeatPayload{TsUnixMs: ts})
		})
		if err != nil && ctx.Err() == nil && err != hbCtx.Err() && err != context.Canceled {
			log.Debug("heartbeat loop exit", "err", err)
		}
	}()

	for ctx.Err() == nil {
		env, err := peer.RecvValidated(ctx)
		if err != nil {
			return fmt.Errorf("ws read: %w", err)
		}
		recordSaasSeq(saasSeq, env.Seq)

		switch env.Type {
		case ws.MsgHeartbeat:
			continue
		case ws.MsgAuth:
			var inline auth.AuthResult
			if json.Unmarshal(env.Payload, &inline) == nil && !inline.OK {
				return fmt.Errorf("inline auth rejection: %s", inline.Message)
			}
			continue
		case ws.MsgReportAck:
			continue
		case ws.MsgCommand:
			cmd, err := decodeCommandPayload(env.Payload)
			var ackSeq = env.Seq
			mu.Lock()
			if err != nil {
				aerr := outbound.Next()
				_ = peer.Send(ctx, ws.MsgCommandAck, aerr, &ackSeq, command.CommandAck{
					CommandID:       "",
					Status:          command.CommandStatusRejected,
					RefEnvelopeSeq:  env.Seq,
					Message:         err.Error(),
					AgentTimeUnixMs: time.Now().UnixMilli(),
				})
				mu.Unlock()
				continue
			}
			ack, rep := execsvc.HandleTradeCommand(ctx, cmd, ackSeq, time.Now().UnixMilli())
			if cerr := peer.Send(ctx, ws.MsgCommandAck, outbound.Next(), &ackSeq, ack); cerr != nil {
				mu.Unlock()
				return fmt.Errorf("command_ack send: %w", cerr)
			}
			if rerr := peer.Send(ctx, ws.MsgDeltaReport, outbound.Next(), nil, rep); rerr != nil {
				mu.Unlock()
				return fmt.Errorf("delta_report send: %w", rerr)
			}
			mu.Unlock()
		default:
			log.Debug("ignoring SaaS envelope", "type", env.Type)
		}
	}
	return ctx.Err()
}

func decodeCommandPayload(raw []byte) (command.TradeCommand, error) {
	var cmd command.TradeCommand
	if err := json.Unmarshal(raw, &cmd); err != nil {
		return command.TradeCommand{}, fmt.Errorf("command payload: %w", err)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		return command.TradeCommand{}, fmt.Errorf("missing command_id")
	}
	return cmd, nil
}

func parseMaxNotional(max string, log *slog.Logger) float64 {
	max = strings.TrimSpace(max)
	if max == "" {
		return 0
	}
	f, err := strconv.ParseFloat(max, 64)
	if err != nil {
		log.Warn("invalid risk.max_notional_quote_per_order; disabling notional ceiling", "err", err)
		return 0
	}
	return f
}

func recordSaasSeq(saasSeq *atomic.Int64, seq int64) {
	if saasSeq == nil || seq <= 0 {
		return
	}
	for {
		old := saasSeq.Load()
		if seq <= old {
			return
		}
		if saasSeq.CompareAndSwap(old, seq) {
			return
		}
	}
}
