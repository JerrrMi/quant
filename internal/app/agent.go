package app

import (
	"context"
	"log/slog"

	"github.com/altshort/quant/internal/config"
	"github.com/altshort/quant/internal/lifecycle"
)

// RunAgent 启动 Agent 占位逻辑：后续在此初始化 WebSocket 客户端、ExchangeExecutor、Heartbeat。
// 由 cmd/agent 的 run() 调用；Binance Key 仅从环境读取，不进入 cfg 结构体。
func RunAgent(ctx context.Context, cfg config.AgentConfig, deps lifecycle.BootstrapDeps) error {
	_ = ctx
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	log.Info("Agent stub ready", "saas_ws", cfg.SaasWSURL, "exchange", cfg.Exchange.Name)
	// TODO: NewWSClient, executor.NewExchangeExecutor, heartbeat.NewManager
	return nil
}
