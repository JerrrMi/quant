package ws_test

import (
	"context"
	"testing"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/auth"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/report"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/ws"
)

type memoryPipe struct {
	incoming chan []byte
	outgoing chan []byte
}

func pairedMemoryPipes(buffer int) (*memoryPipe, *memoryPipe) {
	aToB := make(chan []byte, buffer)
	bToA := make(chan []byte, buffer)
	saasSide := &memoryPipe{incoming: bToA, outgoing: aToB}
	agentSide := &memoryPipe{incoming: aToB, outgoing: bToA}
	return saasSide, agentSide
}

func (m *memoryPipe) ReadFrame(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-m.incoming:
		return append([]byte(nil), data...), nil
	}
}

func (m *memoryPipe) WriteFrame(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.outgoing <- append([]byte(nil), data...):
		return nil
	}
}

func (m *memoryPipe) Close() error { return nil }

func TestMinimalLoopAuthCommandAckDeltaReportAck(t *testing.T) {
	ctx := context.Background()
	saasPipe, agentPipe := pairedMemoryPipes(16)
	saasPeer := ws.NewPeer(ws.RoleSaaS, saasPipe, nil)
	agentPeer := ws.NewPeer(ws.RoleAgent, agentPipe, nil)

	// Agent → SaaS auth
	agentAuth := auth.AuthMessage{
		ProtocolVersion: "ws-protocol-v1",
		ClientID:        "agent-cli",
		Nonce:           "n-once",
	}
	if err := agentPeer.Send(ctx, ws.MsgAuth, 1, nil, agentAuth); err != nil {
		t.Fatal(err)
	}
	env0, err := saasPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotAuth auth.AuthMessage
	if err := ws.DecodePayload(env0, &gotAuth); err != nil {
		t.Fatal(err)
	}
	if gotAuth.ProtocolVersion != agentAuth.ProtocolVersion {
		t.Fatalf("auth mismatch %+v", gotAuth)
	}

	// SaaS → Agent auth result
	authOK := auth.AuthResult{
		OK:               true,
		SessionID:      "sess-1",
		GrantedScopes:    []auth.AuthScope{auth.ScopeExecutionWrite},
		ServerTimeUnixMs: 1700000000000,
	}
	if err := saasPeer.Send(ctx, ws.MsgAuth, 1, nil, authOK); err != nil {
		t.Fatal(err)
	}
	envAuthBack, err := agentPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotResult auth.AuthResult
	if err := ws.DecodePayload(envAuthBack, &gotResult); err != nil || !gotResult.OK {
		t.Fatalf("auth result %+v err %v", gotResult, err)
	}

	// SaaS → Agent command
	cmd := command.TradeCommand{
		CommandID:      "cmd-1",
		InstanceID:     "inst-1",
		StrategyID:     "strat-1",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "i1", Symbol: "BTCUSDT", Side: domain.SideSell},
		ReduceOnly:     false,
		DeadlineUnixMs: 9000000000000,
		Nonce:          "cmd-nonce",
		IdempotencyKey: "idem-alpha",
		Kind:           command.CommandKindPlace,
	}
	n := 120000.5
	cmd.TargetNotional = &n

	if err := saasPeer.Send(ctx, ws.MsgCommand, 2, nil, cmd); err != nil {
		t.Fatal(err)
	}
	envCmd, err := agentPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotCmd command.TradeCommand
	if err := ws.DecodePayload(envCmd, &gotCmd); err != nil {
		t.Fatal(err)
	}

	dedup := ws.NewCommandDedup()
	if !dedup.FirstApply(gotCmd) {
		t.Fatal("expected first apply")
	}

	ackSeq := int64(2)
	ack := command.CommandAck{
		CommandID:       gotCmd.CommandID,
		Status:          command.CommandStatusAccepted,
		RefEnvelopeSeq:  envCmd.Seq,
		AgentTimeUnixMs: 1700000000001,
	}
	if err := agentPeer.Send(ctx, ws.MsgCommandAck, 2, &ackSeq, ack); err != nil {
		t.Fatal(err)
	}
	envAckIn, err := saasPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotAck command.CommandAck
	if err := ws.DecodePayload(envAckIn, &gotAck); err != nil || gotAck.CommandID != cmd.CommandID {
		t.Fatalf("ack %+v %v", gotAck, err)
	}

	// Agent → SaaS delta_report
	rep := report.DeltaReport{
		ReportID:                "rep-1",
		InstanceID:              cmd.InstanceID,
		Fills:                   []report.FillRecord{{FillID: "f1", Symbol: cmd.Symbol, Side: domain.SideSell, Price: 1, Quantity: 2, Fee: 0, ExchangeTradeTimeUnixMs: 99}},
		OpenOrders:              []report.OpenOrderSnapshot{{ExchangeOrderID: "ex1", Symbol: cmd.Symbol, Side: domain.SideSell}},
		Positions:               []report.PositionSnapshot{{Symbol: cmd.Symbol, PositionQty: -2, ExchangePositionTimeUnixMs: 100}},
		Account:                 &report.AccountSnapshot{EquityUSDT: 1000, ExchangeAccountTimeUnixMs: 101},
		Errors:                  []string{"noop-err"},
		ExchangeEventTimeUnixMs: 102,
	}
	if err := agentPeer.Send(ctx, ws.MsgDeltaReport, 3, nil, rep); err != nil {
		t.Fatal(err)
	}
	envRep, err := saasPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotRep report.DeltaReport
	if err := ws.DecodePayload(envRep, &gotRep); err != nil || gotRep.ReportID != rep.ReportID {
		t.Fatalf("report %+v %v", gotRep, err)
	}

	reportAckSeq := int64(3)
	rack := report.ReportAck{
		ReportID:         rep.ReportID,
		Received:         true,
		RefEnvelopeSeq:   envRep.Seq,
		ServerTimeUnixMs: 1700000000002,
	}
	if err := saasPeer.Send(ctx, ws.MsgReportAck, 3, &reportAckSeq, rack); err != nil {
		t.Fatal(err)
	}
	envRack, err := agentPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gotRack report.ReportAck
	if err := ws.DecodePayload(envRack, &gotRack); err != nil || !gotRack.Received {
		t.Fatalf("report ack %+v %v", gotRack, err)
	}
}

func TestReconnectReplayDoesNotReexecuteCommand(t *testing.T) {
	ctx := context.Background()
	dedup := ws.NewCommandDedup()

	cmd := command.TradeCommand{
		CommandID:      "cmd-9",
		InstanceID:     "inst-9",
		StrategyID:     "strat-9",
		Symbol:         "ETHUSDT",
		Side:           domain.SideBuy,
		Intent:         strategy.TradeIntent{IntentID: "z", Symbol: "ETHUSDT", Side: domain.SideBuy},
		DeadlineUnixMs: 9000000000000,
		Nonce:          "x",
		IdempotencyKey: "same-key",
		Kind:           command.CommandKindPlace,
	}

	if !dedup.FirstApply(cmd) {
		t.Fatal("first delivery must apply")
	}
	if dedup.FirstApply(cmd) {
		t.Fatal("replay of same idempotency must not apply twice")
	}

	// Simulate SaaS replay frame after reconnect (same logical command still seq 2).
	saasPipe, agentPipe := pairedMemoryPipes(4)
	saasPeer := ws.NewPeer(ws.RoleSaaS, saasPipe, nil)
	agentPeer := ws.NewPeer(ws.RoleAgent, agentPipe, nil)
	if err := saasPeer.Send(ctx, ws.MsgCommand, 2, nil, cmd); err != nil {
		t.Fatal(err)
	}
	env, err := agentPeer.RecvValidated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var replayed command.TradeCommand
	if err := ws.DecodePayload(env, &replayed); err != nil {
		t.Fatal(err)
	}
	if dedup.FirstApply(replayed) {
		t.Fatal("socket-layer replay must be ignored by dedup")
	}
}

func TestMergeFillSnapshotsIdempotentByFillID(t *testing.T) {
	a := []report.FillRecord{{FillID: "1", Quantity: 1}}
	b := []report.FillRecord{{FillID: "1", Quantity: 999}, {FillID: "2", Quantity: 2}}
	out := ws.MergeFillSnapshots(a, b)
	if len(out) != 2 || out[0].Quantity != 1 || out[1].FillID != "2" {
		t.Fatalf("got %+v", out)
	}
}

// 约束：应答帧必须带 ack_seq，且与对端被确认帧 seq 对齐。
func TestCommandAck_rejectsMissingAckSeq(t *testing.T) {
	ctx := context.Background()
	trans := &memoryPipe{incoming: make(chan []byte, 1), outgoing: make(chan []byte, 1)}
	saasPeer := ws.NewPeer(ws.RoleSaaS, trans, nil)
	ack := command.CommandAck{CommandID: "c", Status: command.CommandStatusAccepted, RefEnvelopeSeq: 7}
	err := saasPeer.Send(ctx, ws.MsgCommandAck, 3, nil, ack)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// 非应答类型禁止携带 ack_seq（防止错误关联）。
func TestCommand_mustNotIncludeAckSeq(t *testing.T) {
	ctx := context.Background()
	trans := &memoryPipe{incoming: make(chan []byte, 1), outgoing: make(chan []byte, 1)}
	saasPeer := ws.NewPeer(ws.RoleSaaS, trans, nil)
	ackRef := int64(9)
	cmd := command.TradeCommand{CommandID: "c1", Symbol: "BTCUSDT", Side: domain.SideSell}
	err := saasPeer.Send(ctx, ws.MsgCommand, 2, &ackRef, cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// JSONCodec 对嵌套领域类型的往返应保持协议稳定（字段由 json tag 锁定）。
func TestJSONCodec_envelopeRoundtrip(t *testing.T) {
	var codec ws.JSONCodec
	cmd := command.TradeCommand{
		CommandID: "id-1", InstanceID: "i", StrategyID: "s", Symbol: "BTCUSDT",
		Side: domain.SideSell, Kind: command.CommandKindPlace, IdempotencyKey: "ik",
	}
	raw, err := codec.MarshalEnvelope(ws.MsgCommand, 5, nil, cmd)
	if err != nil {
		t.Fatal(err)
	}
	env, err := codec.UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != ws.MsgCommand || env.Seq != 5 || env.AckSeq != nil {
		t.Fatalf("envelope shell: %+v", env)
	}
	var got command.TradeCommand
	if err := ws.DecodePayload(env, &got); err != nil {
		t.Fatal(err)
	}
	if got.CommandID != cmd.CommandID || got.IdempotencyKey != cmd.IdempotencyKey || got.Side != cmd.Side {
		t.Fatalf("payload mismatch %+v", got)
	}
}

// 断线后新会话清空进程内去重表时，同键会再次「首次投递」——持久化幂等必须由执行器/DB 底座兜底。
func TestCommandDedup_dropSeenModelsSessionReset(t *testing.T) {
	d := ws.NewCommandDedup()
	cmd := command.TradeCommand{IdempotencyKey: "idem-session"}
	if !d.FirstApply(cmd) {
		t.Fatal("first apply")
	}
	if d.FirstApply(cmd) {
		t.Fatal("duplicate in same session should not apply twice")
	}
	d.DropSeen(cmd)
	if !d.FirstApply(cmd) {
		t.Fatal("after DropSeen, memory layer allows re-delivery; downstream dedup must still protect venue idempotency")
	}
}

// SaaS 缓冲帧是否在 Agent last_seen 之后需要重放。
func TestSaasOutboundNeedsReplay_semantics(t *testing.T) {
	if !ws.SaasOutboundNeedsReplay(3, 4) {
		t.Fatal("strictly newer seq should replay")
	}
	if ws.SaasOutboundNeedsReplay(4, 4) {
		t.Fatal("equal seq is not replay")
	}
}
