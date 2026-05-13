package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/altshort/quant/internal/app"
	"github.com/altshort/quant/internal/config"
	"github.com/altshort/quant/internal/executor"
	"github.com/altshort/quant/internal/infra"
	"github.com/altshort/quant/internal/lifecycle"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("agent exited", "err", err)
		os.Exit(1)
	}
}

// run 加载 Agent 配置、初始化 Logger 与执行依赖占位（WebSocket client、ExchangeExecutor、Heartbeat 在后续 Phase 接入）。
// Binance API Key 仅允许从环境变量读取，禁止写入 YAML 结构体。
func run(ctx context.Context) error {
	slog.Info("AltShort Agent process starting")

	cfgPath := filepath.Clean("configs/agent.yaml")
	cfg, err := config.LoadAgentConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load agent config: %w", err)
	}

	log := infra.NewLogger("agent", cfg.Logging.Level)

	// Agent 侧数据库可选；骨架阶段使用内存 SQLite 以便预留 AutoMigrate 钩子。
	db, err := infra.OpenSQLite("file::memory:?cache=shared")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	deps := lifecycle.NewBootstrapDeps(log, db)

	// --- 预留：WebSocket client 连接 cfg.SaasWSURL ---
	// --- 预留：executor.NewExchangeExecutor(cfg.Exchange.Name)；密钥从 os.Getenv 注入 ---
	ex := executor.NewExchangeExecutor(cfg.Exchange.Name)
	if err := ex.Ping(ctx); err != nil {
		return fmt.Errorf("executor ping: %w", err)
	}
	// --- 预留：heartbeat manager 与重连策略 ---

	log.Info("Agent bootstrap complete", "config", cfgPath, "saas_ws", cfg.SaasWSURL, "executor", ex.Venue)

	if err := app.RunAgent(ctx, cfg, deps); err != nil {
		return fmt.Errorf("app.RunAgent: %w", err)
	}
	return nil
}
