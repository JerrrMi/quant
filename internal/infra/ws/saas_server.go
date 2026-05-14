package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/JerrrMi/quant/internal/domain/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

// SaasAgentServer 在挂载路径上接受 Agent WebSocket，完成最小 auth 注册到 Hub。
type SaasAgentServer struct {
	Hub    *AgentHub
	Logger *slog.Logger
}

func (s *SaasAgentServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.Hub == nil {
		http.Error(w, "hub unavailable", http.StatusServiceUnavailable)
		return
	}
	log := s.Logger
	if log == nil {
		log = slog.Default()
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn("ws upgrade failed", "err", err)
		return
	}
	go s.handleConn(r.Context(), conn, log)
}

func (s *SaasAgentServer) handleConn(base context.Context, conn *websocket.Conn, log *slog.Logger) {
	defer conn.Close()
	transport := &WebsocketTransport{Conn: conn}
	peer := NewPeer(RoleSaaS, transport, nil)

	ctx, cancel := context.WithCancel(base)
	defer cancel()

	env, err := peer.RecvValidated(ctx)
	if err != nil || env == nil {
		return
	}
	if env.Type != MsgAuth {
		log.Warn("ws first frame not auth", "type", env.Type)
		return
	}
	var msg auth.AuthMessage
	if err := json.Unmarshal(env.Payload, &msg); err != nil {
		log.Warn("ws auth decode failed", "err", err)
		return
	}
	if msg.ClientID == "" {
		_ = sendAuthErr(ctx, peer, 1, "client_id_required")
		return
	}
	res := auth.AuthResult{
		OK:               true,
		ServerTimeUnixMs: time.Now().UTC().UnixMilli(),
		SessionID:        msg.Nonce,
		GrantedScopes:    append([]auth.AuthScope(nil), msg.RequestedScopes...),
	}
	if err := peer.Send(ctx, MsgAuth, 1, nil, res); err != nil {
		log.Warn("ws auth reply failed", "err", err)
		return
	}
	unreg := s.Hub.Register(msg.ClientID, peer, 1)
	defer unreg()

	// 读循环：消耗 Agent 侧心跳/回报，避免死读缓冲阻塞；业务处理可后续接入。
	for {
		env, err := peer.RecvValidated(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Debug("ws read ended", "agent", msg.ClientID, "err", err)
			}
			return
		}
		switch env.Type {
		case MsgHeartbeat, MsgCommandAck, MsgDeltaReport:
			// 占位：调度与同步后续扩展
		default:
			log.Debug("ws ignored inbound", "type", env.Type)
		}
	}
}

func sendAuthErr(ctx context.Context, peer *Peer, seq int64, code string) error {
	res := auth.AuthResult{
		OK:               false,
		ErrorCode:        code,
		Message:          code,
		ServerTimeUnixMs: time.Now().UTC().UnixMilli(),
	}
	return peer.Send(ctx, MsgAuth, seq, nil, res)
}
