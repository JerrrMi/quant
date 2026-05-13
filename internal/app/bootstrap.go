// Bootstrap：按进程装配 BootstrapDeps（stub），不含具体业务编排。

package app

import (
	"fmt"
	"log/slog"

	"github.com/altshort/quant/internal/config"
	"github.com/altshort/quant/internal/infra"
	"gorm.io/gorm"
)

// BootstrapDeps 是进程运行时侧效应依赖的挂载点。
// 由 Bootstrap* 函数填充占位字段，供 Run*、HTTP/WebSocket、调度等后续 Phase 使用。
type BootstrapDeps struct {
	// Logger 为进程级日志句柄。
	Logger *slog.Logger
	// DB 为可选 GORM 连接；不需持久化时仍为 nil。
	DB *gorm.DB
	// Cache 为未来 Redis/Memcached 客户端占位；当前通常为 nil。
	Cache any
	// WSServer 为 SaaS 侧 WebSocket 服务端占位。
	WSServer any
	// WSClient 为 Agent 侧 WebSocket 客户端占位。
	WSClient any
	// Executor 为交易所执行适配占位（仅 Agent 有意义）。
	Executor any
	// Scheduler 为未来定时/节拍调度器占位。
	Scheduler any
}

// BootstrapSaaS 输入：已通过 Validate 的 SaaSConfig、由调用方创建的 Logger。
// 输出：BootstrapDeps（含打开的 DB）。
// 后续由 app.RunSaaS、控制面 HTTP/WS 层消费。
func BootstrapSaaS(cfg config.SaaSConfig, logger *slog.Logger) (BootstrapDeps, error) {
	if logger == nil {
		return BootstrapDeps{}, fmt.Errorf("bootstrap saas: logger is nil")
	}
	dsn := cfg.Database.DSN
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	db, err := infra.OpenSQLite(dsn)
	if err != nil {
		return BootstrapDeps{}, fmt.Errorf("bootstrap saas: open database: %w", err)
	}
	return BootstrapDeps{
		Logger:    logger,
		DB:        db,
		Cache:     nil,
		WSServer:  nil,
		WSClient:  nil,
		Executor:  nil,
		Scheduler: nil,
	}, nil
}

// BootstrapAgent 输入：AgentConfig（已校验）、Logger。
// 输出：Deps；含本地 SQLite（内存或可配置路径），不含真实 executor 连通性探测。
// 后续由 app.RunAgent、executor/心跳/WebSocket client 装配使用。
func BootstrapAgent(cfg config.AgentConfig, logger *slog.Logger) (BootstrapDeps, error) {
	if logger == nil {
		return BootstrapDeps{}, fmt.Errorf("bootstrap agent: logger is nil")
	}
	dsn := cfg.LocalStore.SQLiteDSN
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	db, err := infra.OpenSQLite(dsn)
	if err != nil {
		return BootstrapDeps{}, fmt.Errorf("bootstrap agent: open local sqlite: %w", err)
	}
	return BootstrapDeps{
		Logger:    logger,
		DB:        db,
		Cache:     nil,
		WSServer:  nil,
		WSClient:  nil,
		Executor:  nil,
		Scheduler: nil,
	}, nil
}

// BootstrapBacktest 输入：BacktestConfig（已校验）、Logger。
// 输出：Deps；回放引擎与历史加载器仍在 RunBacktest 中挂载，以保持 bootstrap 无副作用业务。
// 后续由 internal/backtest 与策略 Step 链路消费。
func BootstrapBacktest(cfg config.BacktestConfig, logger *slog.Logger) (BootstrapDeps, error) {
	_ = cfg
	if logger == nil {
		return BootstrapDeps{}, fmt.Errorf("bootstrap backtest: logger is nil")
	}
	return BootstrapDeps{
		Logger:    logger,
		DB:        nil,
		Cache:     nil,
		WSServer:  nil,
		WSClient:  nil,
		Executor:  nil,
		Scheduler: nil,
	}, nil
}
