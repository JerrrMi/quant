// Package db 集中管理 GORM 迁移入口；禁止引入 SQL migration 文件。
package db

import (
	"fmt"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// AutoMigrateAll 对下列模型执行 AutoMigrate；启动时调用，失败返回可包装的错误。
// 模型列表变更时请同步更新 docs/data-ownership.md。
func AutoMigrateAll(g *gorm.DB) error {
	if g == nil {
		return fmt.Errorf("db automigrate: gorm db is nil")
	}

	// 显式枚举迁移模型，避免反射式「扫全包」导致顺序与遗漏不透明。
	toMigrate := []any{
		&models.User{},
		&models.Instance{},
		&models.Strategy{},
		&models.StrategyRun{},
		&models.MarketSnapshot{},
		&models.LPPLScanResult{},
		&models.TradeCommandRecord{},
		&models.TradeFillRecord{},
		&models.AgentReportRecord{},
		&models.WSSession{},
		&models.AuditEvent{},
		&models.AgentDedupKey{},
	}

	if err := g.AutoMigrate(toMigrate...); err != nil {
		return fmt.Errorf("db automigrate: %w", err)
	}
	return nil
}
