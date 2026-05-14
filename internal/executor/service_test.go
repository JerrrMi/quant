package executor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/report"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/executor"
	"github.com/JerrrMi/quant/internal/infra"
	"github.com/JerrrMi/quant/internal/infra/agentstate"
	"github.com/JerrrMi/quant/internal/infra/binance"
	"github.com/JerrrMi/quant/internal/infra/db"
)

func TestService_NoVenue(t *testing.T) {
	var s executor.Service
	now := time.Now().UnixMilli()
	ack, rep := s.HandleTradeCommand(context.Background(), command.TradeCommand{CommandID: "x"}, 1, now)
	if ack.Status != command.CommandStatusRejected || rep.ReportID == "" {
		t.Fatalf("ack %+v rep %+v", ack, rep)
	}
}

type stubVenue struct{}

func (stubVenue) SyncTime(context.Context) error                     { return nil }
func (stubVenue) MarkPrice(context.Context, string) (float64, error) { return 50000, nil }
func (stubVenue) LotStep(context.Context, string) (float64, error)   { return 0.001, nil }

func (stubVenue) PlaceMarket(context.Context, executor.PlaceMarketIn) (*executor.OrderView, error) {
	return &executor.OrderView{VenueOrderID: 42, Symbol: "BTCUSDT", Status: "FILLED"}, nil
}

func (stubVenue) Cancel(context.Context, executor.CancelIn) (*executor.OrderView, error) {
	return nil, nil
}

func (stubVenue) QueryByClientOrder(context.Context, string, string) (*executor.OrderView, error) {
	return nil, nil
}

func (stubVenue) QueryByVenueOrderID(context.Context, string, int64) (*executor.OrderView, error) {
	return nil, nil
}

func (stubVenue) UserTrades(context.Context, string, int64) ([]report.FillRecord, error) {
	return nil, nil
}

func (stubVenue) AccountAndPositions(context.Context) (*executor.VenueAccountView, error) {
	return &executor.VenueAccountView{}, nil
}

func (stubVenue) OpenOrders(context.Context, string) ([]executor.OpenOrderView, error) {
	return nil, nil
}

func TestService_CommandExpiredDeadline(t *testing.T) {
	svc := executor.NewService(stubVenue{}, nil, 0, 0)
	not := 1000.0
	cmd := command.TradeCommand{
		CommandID:      "c1",
		InstanceID:     "i",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "a"},
		TargetNotional: &not,
		DeadlineUnixMs: 100,
		IdempotencyKey: "idem",
		Kind:           command.CommandKindPlace,
	}
	ack, rep := svc.HandleTradeCommand(context.Background(), cmd, 1, 200000)
	if ack.Status != command.CommandStatusExpired {
		t.Fatalf("expected expired, got %+v %+v", ack, rep)
	}
}

// 约束：相同 IdempotencyKey 重放不得再次触发 PlaceMarket（依赖本地 Dedup + QueryByClientOrder 对齐）。
func TestService_idempotentReplaySkipsSecondPlace(t *testing.T) {
	g, err := infra.OpenSQLite("file::memory:?cache=private")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrateAll(g); err != nil {
		t.Fatal(err)
	}
	dedup := agentstate.NewDedupStore(g)

	var v countingVenue
	svc := executor.NewService(&v, dedup, 0, 0)
	not := 800.0
	base := command.TradeCommand{
		InstanceID:     "inst",
		StrategyID:     "strat",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "a1"},
		TargetNotional: &not,
		DeadlineUnixMs: 9_000_000_000_000,
		IdempotencyKey:   "corr-idem-1",
		Kind:           command.CommandKindPlace,
	}
	ctx := context.Background()

	cmd1 := base
	cmd1.CommandID = "first-id"
	ack1, _ := svc.HandleTradeCommand(ctx, cmd1, 1, 1000)
	if ack1.Status == command.CommandStatusRejected {
		t.Fatalf("first cmd rejected: %+v", ack1)
	}
	if v.placeCalls != 1 {
		t.Fatalf("placeCalls=%d", v.placeCalls)
	}

	cmd2 := base
	cmd2.CommandID = "second-id-replay"
	ack2, _ := svc.HandleTradeCommand(ctx, cmd2, 2, 1001)
	if ack2.Status == command.CommandStatusRejected && ack2.Message == "" {
		t.Fatalf("unexpected reject %+v", ack2)
	}
	if v.placeCalls != 1 {
		t.Fatalf("second Handle should not Place again, placeCalls=%d", v.placeCalls)
	}
	if v.queryCalls < 1 {
		t.Fatalf("expected venue query on replay, queryCalls=%d", v.queryCalls)
	}
}

type countingVenue struct {
	stubVenue
	mu         sync.Mutex
	placeCalls int
	queryCalls int
}

func (v *countingVenue) PlaceMarket(ctx context.Context, in executor.PlaceMarketIn) (*executor.OrderView, error) {
	v.mu.Lock()
	v.placeCalls++
	v.mu.Unlock()
	return v.stubVenue.PlaceMarket(ctx, in)
}

func (v *countingVenue) QueryByClientOrder(ctx context.Context, symbol, clientOID string) (*executor.OrderView, error) {
	v.mu.Lock()
	v.queryCalls++
	v.mu.Unlock()
	if clientOID == "" {
		return nil, nil
	}
	return &executor.OrderView{VenueOrderID: 9001, Symbol: symbol, Status: "FILLED", ClientOrderID: clientOID}, nil
}

// 约束：可重试 API 错误在 bounded backoff 后成功（RetryAfter 取极小值避免慢测）。
func TestService_placeRetriesOnRetryableAPIError(t *testing.T) {
	var v retryOnceVenue
	svc := executor.NewService(&v, nil, 0, 0)
	not := 600.0
	cmd := command.TradeCommand{
		CommandID:      "cid",
		InstanceID:     "i",
		StrategyID:     "s",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "iid"},
		TargetNotional: &not,
		DeadlineUnixMs: 9_000_000_000_000,
		IdempotencyKey: "idem-retry",
		Kind:           command.CommandKindPlace,
	}
	ack, _ := svc.HandleTradeCommand(context.Background(), cmd, 1, 1000)
	if ack.Status != command.CommandStatusCompleted {
		t.Fatalf("expected success after retry, got %+v placeCalls=%d", ack, v.placeCalls())
	}
	if v.placeCalls() < 2 {
		t.Fatalf("expected at least 2 place attempts, got %d", v.placeCalls())
	}
}

type retryOnceVenue struct {
	stubVenue
	mu       sync.Mutex
	attempts int
}

func (v *retryOnceVenue) placeCalls() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.attempts
}

func (v *retryOnceVenue) PlaceMarket(ctx context.Context, in executor.PlaceMarketIn) (*executor.OrderView, error) {
	v.mu.Lock()
	v.attempts++
	n := v.attempts
	v.mu.Unlock()
	if n == 1 {
		return nil, &binance.APIError{Retryable: true, RetryAfter: time.Microsecond}
	}
	return (&stubVenue{}).PlaceMarket(ctx, in)
}

// ctx 取消须快速失败，不得依赖真实交易所。
func TestService_contextCancelBeforeVenueWork(t *testing.T) {
	svc := executor.NewService(ctxCancelVenue{}, nil, 0, 0)
	not := 500.0
	cmd := command.TradeCommand{
		CommandID:      "x",
		InstanceID:     "i",
		StrategyID:     "s",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "z"},
		TargetNotional: &not,
		DeadlineUnixMs: 9_000_000_000_000,
		IdempotencyKey: "idem-ctx",
		Kind:           command.CommandKindPlace,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ack, rep := svc.HandleTradeCommand(ctx, cmd, 1, 1000)
	if ack.Status != command.CommandStatusRejected {
		t.Fatalf("expected reject on cancel, got %+v", ack)
	}
	if ack.Message == "" && len(rep.Errors) == 0 {
		t.Fatal("expected reject detail on ack or errors on report")
	}
}

type ctxCancelVenue struct{ stubVenue }

func (ctxCancelVenue) LotStep(ctx context.Context, _ string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return 0.001, nil
}

// DeltaReport 应聚合 Account/仓位视图（refreshSnapshot），与命令结果一并上报。
func TestService_deltaReportIncludesAccountSnapshot(t *testing.T) {
	svc := executor.NewService(snapshotVenue{}, nil, 0, 0)
	not := 400.0
	cmd := command.TradeCommand{
		CommandID:      "snap",
		InstanceID:     "i2",
		StrategyID:     "s2",
		Symbol:         "BTCUSDT",
		Side:           domain.SideSell,
		Intent:         strategy.TradeIntent{IntentID: "q"},
		TargetNotional: &not,
		DeadlineUnixMs: 9_000_000_000_000,
		IdempotencyKey: "idem-snap",
		Kind:           command.CommandKindPlace,
	}
	_, rep := svc.HandleTradeCommand(context.Background(), cmd, 1, 1000)
	if rep.Account == nil {
		t.Fatal("expected account snapshot on report")
	}
	if rep.Account.EquityUSDT != 31415.9 {
		t.Fatalf("equity got %v", rep.Account.EquityUSDT)
	}
}

type snapshotVenue struct{ stubVenue }

func (snapshotVenue) AccountAndPositions(context.Context) (*executor.VenueAccountView, error) {
	return &executor.VenueAccountView{
		EquityUSDT:           31415.9,
		WalletBalanceUSDT:    30000,
		AvailableBalanceUSDT: 29000,
		ExchangeTimeUnixMs:   555,
	}, nil
}
