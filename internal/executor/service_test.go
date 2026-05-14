package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/report"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/executor"
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
