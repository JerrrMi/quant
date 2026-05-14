package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/JerrrMi/quant/internal/app"
	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/infra"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("agent exited with error", "err", err)
		os.Exit(1)
	}
}

// run：加载 Agent 配置 → 校验 → 日志 → 依赖装配 → 应用入口。
func run(ctx context.Context) error {
	slog.Info("AltShort Agent process starting")

	cfgPath := filepath.Clean("configs/agent.yaml")

	cfg, err := config.LoadAgentConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load Agent configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate Agent configuration from %q: %w", cfgPath, err)
	}

	logger := infra.NewLogger("agent", cfg.Logging.Level)

	deps, err := app.BootstrapAgent(cfg, logger)
	if err != nil {
		return fmt.Errorf("assemble Agent dependencies (config=%q): %w", cfgPath, err)
	}

	log := deps.Logger
	log.Info(
		"Agent bootstrap complete",
		"config", cfgPath,
		"saas_ws", cfg.Connection.SaasWSURL,
		"exchange", cfg.Exchange.Name,
	)

	if err := app.RunAgent(ctx, cfg, deps); err != nil {
		return fmt.Errorf("run Agent application: %w", err)
	}
	return nil
}
