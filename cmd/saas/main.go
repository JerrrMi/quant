package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/altshort/quant/internal/app"
	"github.com/altshort/quant/internal/config"
	"github.com/altshort/quant/internal/infra"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("saas exited with error", "err", err)
		os.Exit(1)
	}
}

// run：加载 SaaS 配置 → 校验 → 日志 → 依赖装配 → 应用入口。
func run(ctx context.Context) error {
	slog.Info("AltShort SaaS process starting")

	cfgPath := filepath.Clean("configs/saas.yaml")

	cfg, err := config.LoadSaaSConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load SaaS configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate SaaS configuration from %q: %w", cfgPath, err)
	}

	logger := infra.NewLogger("saas", cfg.Logging.Level)

	deps, err := app.BootstrapSaaS(cfg, logger)
	if err != nil {
		return fmt.Errorf("assemble SaaS dependencies (config=%q): %w", cfgPath, err)
	}

	log := deps.Logger
	log.Info(
		"SaaS bootstrap complete",
		"config", cfgPath,
		"listen", cfg.WebSocket.ListenAddr,
		"agent_ws_path", cfg.WebSocket.AgentPath,
	)

	if err := app.RunSaaS(ctx, cfg, deps); err != nil {
		return fmt.Errorf("run SaaS application: %w", err)
	}
	return nil
}
