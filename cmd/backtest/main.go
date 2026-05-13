package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/JerrrMi/quant/internal/app"
	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/infra"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("backtest exited with error", "err", err)
		os.Exit(1)
	}
}

// run：加载回测配置 → 校验 → 日志 → 依赖装配 → 应用入口。
func run(ctx context.Context) error {
	slog.Info("AltShort Backtest process starting")

	cfgPath := filepath.Clean("configs/backtest.yaml")

	cfg, err := config.LoadBacktestConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load Backtest configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate Backtest configuration from %q: %w", cfgPath, err)
	}

	logger := infra.NewLogger("backtest", cfg.Logging.Level)

	deps, err := app.BootstrapBacktest(cfg, logger)
	if err != nil {
		return fmt.Errorf("assemble Backtest dependencies (config=%q): %w", cfgPath, err)
	}

	log := deps.Logger
	log.Info(
		"Backtest bootstrap complete",
		"config", cfgPath,
		"symbol", cfg.Data.Symbol,
		"capital", cfg.Capital.InitialQuote,
		"currency", cfg.Capital.Currency,
	)

	if err := app.RunBacktest(ctx, cfg, deps); err != nil {
		return fmt.Errorf("run Backtest application: %w", err)
	}
	return nil
}
