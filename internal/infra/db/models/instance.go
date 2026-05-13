package models

import (
	"time"

	"gorm.io/gorm"
)

// Instance 将「用户 + 策略定义」绑定到具体 Agent 部署实例（编排单元）。
// 真源：SaaS（谁跑什么、跑在哪一类 Agent 上，由控制面编排）。
// 软删除：支持（实例退役后仍可从历史命令/回报追溯）。
type Instance struct {
	ID           uint           `gorm:"primaryKey"`
	UserID       uint           `gorm:"index;not null"`
	StrategyID   uint           `gorm:"index;not null"`
	AgentKey     string         `gorm:"size:128;index:idx_instance_agent_key;not null"` // 公开标识，非 API Secret
	DisplayName  string         `gorm:"size:191"`
	Status       string         `gorm:"size:32;index;not null;default:active"` // active|paused|draining
	LastHeartbeatAt *time.Time `gorm:"index"`
	CreatedAt    time.Time      `gorm:"index"`
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
