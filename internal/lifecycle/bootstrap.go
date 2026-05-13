// Package lifecycle 提供进程启动时的依赖束（BootstrapDeps）组装。
// cmd/* 在 run() 中构造 BootstrapDeps 并交给 internal/app；不负责业务用例细节。
package lifecycle

import (
	"log/slog"

	"gorm.io/gorm"
)

// BootstrapDeps 是入口进程已初始化的侧效应依赖集合。
// 由 NewBootstrapDeps 注入 Logger、DB 等；internal/app 接收后编排各服务。
type BootstrapDeps struct {
	// Logger 进程级日志；构造自 infra，供各层以同一 handler 输出。
	Logger *slog.Logger

	// DB 是 GORM 句柄；后续 Phase 在各模块调用 AutoMigrate，不用 SQL migration 文件。
	DB *gorm.DB
}

// NewBootstrapDeps 返回启动依赖束。
// 调用方：cmd/saas、cmd/agent（cmd/backtest 可传 nil DB 若仅回放内存数据）。
// db 可为 nil（例如回测纯内存路径）；调用方需在使用前判断。
func NewBootstrapDeps(logger *slog.Logger, db *gorm.DB) BootstrapDeps {
	return BootstrapDeps{Logger: logger, DB: db}
}
