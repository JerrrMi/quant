package app

import (
	"context"

	agentapp "github.com/JerrrMi/quant/internal/app/agent"
	"github.com/JerrrMi/quant/internal/config"
)

// RunAgent 启动 Agent：WebSocket 接入 SaaS，执行交易所指令并上报回报。
// Binance Key / Secret 仅从 cfg 所指环境变量读取，不进入明文配置。
func RunAgent(ctx context.Context, cfg config.AgentConfig, deps BootstrapDeps) error {
	return agentapp.Run(ctx, cfg, agentapp.Deps{Logger: deps.Logger, DB: deps.DB})
}
