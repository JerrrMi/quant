// Package app 为用例编排层：衔接配置、BootstrapDeps 与各子系统。
// HTTP、WS、调度等在此包或其子目录中逐步落地；cmd 仅调用 Run* 入口。
package app

import (
	"context"
	"log/slog"

	"github.com/JerrrMi/quant/internal/config"
)

// RunSaaS 启动 SaaS 控制面占位逻辑：后续在此挂载 HTTP 与 AutoMigrate。
// 由 cmd/saas 的 run() 在加载配置与 BootstrapDeps 后调用。
func RunSaaS(ctx context.Context, cfg config.SaaSConfig, deps BootstrapDeps) error {
	_ = ctx
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	log.Info("SaaS stub ready", "listen", cfg.WebSocket.ListenAddr, "ws_path", cfg.WebSocket.AgentPath, "has_db", deps.DB != nil)
	// TODO: wire HTTP server, GORM AutoMigrate on deps.DB
	return nil
}
