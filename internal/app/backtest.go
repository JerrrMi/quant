package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/JerrrMi/quant/internal/backtest"
	"github.com/JerrrMi/quant/internal/config"
)

// RunBacktest loads history, walks bars with shared strategy.Step, and logs KPIs.
func RunBacktest(ctx context.Context, cfg config.BacktestConfig, deps BootstrapDeps) error {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	rep, err := backtest.BacktestFromConfig(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("backtest: %w", err)
	}
	m := rep.Metrics
	log.Info("backtest completed",
		"final_equity", m.FinalEquity,
		"total_return", m.TotalReturn,
		"max_drawdown_01", m.MaxDrawdown01,
		"win_rate", m.WinRate,
		"round_trips", m.NumRoundTrips,
		"turnover_ratio", m.TurnoverRatio,
		"command_hit_rate", m.CommandHitRate,
		"command_fail_rate", m.CommandFailRate,
		"partial_fill_rate", m.PartialFillRate,
		"steps", len(rep.EquityCurve),
		"commands", len(rep.CommandStats),
	)
	return nil
}
