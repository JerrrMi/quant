// Package app 为用例编排层：衔接配置、BootstrapDeps 与各子系统。
// HTTP、WS、调度等在此包或其子目录中逐步落地；cmd 仅调用 Run* 入口。
package app

import (
	"context"

	"github.com/JerrrMi/quant/internal/app/saas"
	"github.com/JerrrMi/quant/internal/config"
)

// RunSaaS 启动 SaaS 控制面：WS Hub、调度与数据管线占位。
// 由 cmd/saas 的 run() 在加载配置与 BootstrapDeps 后调用。
func RunSaaS(ctx context.Context, cfg config.SaaSConfig, deps BootstrapDeps) error {
	return saas.Run(ctx, cfg, saas.Deps{Logger: deps.Logger, DB: deps.DB, Cache: deps.Cache})
}
