package backtest

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
)

func TestBacktestFromConfig_runsOneCycle(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "BTCUSDT.csv")
	var b strings.Builder
	b.WriteString("timestamp_ms,open,high,low,close,volume\n")
	ts := int64(1_700_000_000_000)
	for i := 0; i < 80; i++ {
		c := 100.0 + float64(i)*0.01
		_, _ = fmt.Fprintf(&b, "%d,%g,%g,%g,%g,1\n", ts+int64(i)*60_000, c, c, c, c)
	}
	if err := os.WriteFile(csvPath, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	var cfg config.BacktestConfig
	cfg.Data.Provider = "file"
	cfg.Data.Path = csvPath
	cfg.Data.Symbol = "BTCUSDT"
	cfg.Fees.MakerBps = 2
	cfg.Fees.TakerBps = 5
	cfg.Slippage.Bps = 1
	cfg.Replay.WarmupBars = 25
	cfg.Capital.InitialQuote = "100000"
	cfg.Capital.Currency = "USDT"
	cfg.Logging.Level = "info"
	cfg.Model.ID = "test"
	cfg.Model.Values = map[string]float64{"signal_lookback": 10}
	cfg.Simulation.RNGSeed = 7
	cfg.Simulation.LotStep = 0.001

	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	rep, err := BacktestFromConfig(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.EquityCurve) == 0 {
		t.Fatal("expected equity points")
	}
	last := rep.EquityCurve[len(rep.EquityCurve)-1]
	if !finiteOK(last.Equity) || last.Equity <= 0 {
		t.Fatalf("bad final equity %v", last.Equity)
	}
}

func finiteOK(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func TestBuildInput_usesSharedStep(t *testing.T) {
	feed := &DataFeed{
		Symbol:   "BTCUSDT",
		FeatSpec: marketdata.DefaultFeatureWindowSpec(5),
	}
	bars := make([]domain.Bar, 6)
	for i := range bars {
		bars[i] = domain.Bar{Open: 10, High: 10, Low: 10, Close: 10, Volume: 1, TimestampUnixMs: int64(i + 1)}
	}
	in, err := feed.BuildInput(context.Background(), bars, 5, 0, 0, 1, strategy.RiskSnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strategy.Step(in); err != nil {
		t.Fatal(err)
	}
}
