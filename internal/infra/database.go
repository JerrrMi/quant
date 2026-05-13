// Package infra 放置日志、数据库、第三方 API 客户端等可替换基础设施。
// 策略包（domain 纯函数）不得依赖本包。
package infra

import (
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// NewLogger 创建带组件名前缀的 slog 文本日志，输出到 stderr。
// 调用方：cmd/*/run 在加载配置后调用，注入 BootstrapDeps。
func NewLogger(component string, level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	return slog.New(h).With("component", component)
}

// OpenSQLite 使用纯 Go SQLite 驱动打开 dsn（可 :memory: 或文件路径）。
// 调用方：cmd/saas/run 等；后续在此集中设置 GORM 会话选项与连接池。
func OpenSQLite(dsn string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}
