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
	// Symbol 为实例绑定的交易标的（venue 规范化，如 BTCUSDT）；调度与展示优先使用该字段。
	Symbol string `gorm:"size:64;index"`
	// MarketKind：spot | futures。
	MarketKind string `gorm:"size:16;index"`
	// RunMode：backtest | paper | live（控制台语义；策略 Step() 仍为纯函数）。
	RunMode string `gorm:"size:16;index"`
	// ParamsJSON 保存实例侧参数（资金、风控、杠杆、仓位限制等），不经由策略包解析。
	ParamsJSON string `gorm:"type:text"`
	Status       string         `gorm:"size:32;index;not null;default:active"` // active|paused|draining
	LastHeartbeatAt *time.Time `gorm:"index"`
	CreatedAt    time.Time      `gorm:"index"`
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
