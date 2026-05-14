package scheduler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra"
	infradb "github.com/JerrrMi/quant/internal/infra/db"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
	"gorm.io/gorm"
)

type stubStepper struct{}

func (stubStepper) Step(in strategy.AltShortStrategyInput) (strategy.AltShortStrategyOutput, error) {
	return strategy.AltShortStrategyOutput{
		Intents: []strategy.TradeIntent{{
			IntentID: "itest",
			Symbol:   in.Symbol,
			Side:     domain.SideSell,
		}},
	}, nil
}

func TestStepOrchestrator_Tick_persistsCommand(t *testing.T) {
	gdb, err := infra.OpenSQLite("file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sdb, _ := gdb.DB()
		_ = sdb.Close()
	})
	if err := infradb.AutoMigrateAll(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seedInstanceAndBars(t, gdb, "BTCUSDT"); err != nil {
		t.Fatal(err)
	}

	orch := &StepOrchestrator{
		Stepper:    stubStepper{},
		Bars:       &marketdata.DBBarSeriesReader{DB: gdb},
		Runs:       repository.NewGormStrategyRunRepository(gdb),
		Instances:  repository.NewGormInstanceRepository(gdb),
		Commands:   repository.NewGormCommandRepository(gdb),
		Audit:      repository.NewGormAuditRepository(gdb),
		Dispatcher: nil,
		Model: config.ModelParams{
			Values: map[string]float64{"signal_lookback": 8},
		},
		DefaultSym: "BTCUSDT",
		Deadline:   time.Minute,
	}
	if err := orch.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}

	var rows []models.TradeCommandRecord
	if err := gdb.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("commands %d", len(rows))
	}
	if rows[0].Status != "pending" {
		t.Fatalf("status %s", rows[0].Status)
	}
	var audit []models.AuditEvent
	if err := gdb.Where("action = ?", "strategy.step_input").Find(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if len(audit) != 1 {
		t.Fatalf("audit inputs %d", len(audit))
	}
	var in strategy.AltShortStrategyInput
	if err := json.Unmarshal([]byte(audit[0].PayloadJSON), &in); err != nil {
		t.Fatal(err)
	}
	if in.Symbol != "BTCUSDT" || in.StepSequence != 1 {
		t.Fatalf("%+v", in)
	}
}

func seedInstanceAndBars(t *testing.T, gdb *gorm.DB, symbol string) error {
	t.Helper()
	u := models.User{Email: "u@test.local", DisplayName: "u"}
	if err := gdb.Create(&u).Error; err != nil {
		return err
	}
	s := models.Strategy{UserID: u.ID, Name: "n", Kind: "k", ConfigJSON: "{}"}
	if err := gdb.Create(&s).Error; err != nil {
		return err
	}
	inst := models.Instance{UserID: u.ID, StrategyID: s.ID, AgentKey: "agent-x", Status: "active"}
	if err := gdb.Create(&inst).Error; err != nil {
		return err
	}
	p1 := marketdata.SnapshotPayload{
		Bar: domain.Bar{Open: 100, High: 101, Low: 99, Close: 100, Volume: 1, TimestampUnixMs: 1000},
	}
	p2 := marketdata.SnapshotPayload{
		Bar: domain.Bar{Open: 100, High: 102, Low: 98, Close: 101, Volume: 2, TimestampUnixMs: 2000},
	}
	b1, _ := json.Marshal(p1)
	b2, _ := json.Marshal(p2)
	now := time.Now().UTC()
	for _, payload := range [][]byte{b1, b2} {
		row := models.MarketSnapshot{
			Symbol:      symbol,
			CapturedAt:  now,
			Source:      "test",
			PayloadJSON: string(payload),
		}
		if err := gdb.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
