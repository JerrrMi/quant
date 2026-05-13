// Package models 定义 GORM 表结构；不含业务规则与策略计算。
package models

import (
	"time"

	"gorm.io/gorm"
)

// User 控制面用户与租户根实体。
// 真源：SaaS（认证与授权在控制面统一裁决；Agent 进程不持有用户主数据）。
// 软删除：支持（运营侧停用账户时保留审计关联）。
type User struct {
	ID           uint           `gorm:"primaryKey"`
	ExternalSub  *string        `gorm:"size:128;uniqueIndex"` // OIDC subject 等；nil 表示无外部主体（SQLite 允许多 NULL 于 UNIQUE）
	Email        string         `gorm:"size:191;uniqueIndex"`
	DisplayName  string         `gorm:"size:191"`
	CreatedAt    time.Time      `gorm:"index"`
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
