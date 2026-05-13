package models

import (
	"time"

	"gorm.io/gorm"
)

// Strategy 策略定义与元数据（名称、类型、版本化参数快照）。
// 真源：SaaS（控制面注册的策略 catalog；Agent 仅按 Instance 绑定引用，不私自篡改定义）。
// 软删除：支持（下架策略时保留历史运行与审计）。
type Strategy struct {
	ID          uint           `gorm:"primaryKey"`
	UserID      uint           `gorm:"index:idx_strategy_user_name,priority:1;not null"`
	Name        string         `gorm:"size:128;index:idx_strategy_user_name,priority:2;not null"`
	Kind        string         `gorm:"size:64;index;not null"` // 策略类型标识
	Version     int            `gorm:"not null;default:1"`
	ConfigJSON  string         `gorm:"type:text;not null"` // JSON：参数快照，策略包仍保持纯函数
	Description string         `gorm:"size:512"`
	CreatedAt   time.Time      `gorm:"index"`
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
