package models

import (
	"time"
)

// TradeCommandRecord 发往 Agent 的交易指令（及其在控制面的状态机）。
// 真源：SaaS 库中为「指令发布与审计」真源；Agent 本地库可为镜像副本用于断线恢复（以 SaaS 重放/对账为准，冲突时 SaaS 裁决）。
// 软删除：不支持（指令为不可变事件；撤单用新指令或状态迁移）。
type TradeCommandRecord struct {
	ID             string    `gorm:"type:char(36);primaryKey"` // UUID
	CorrelationID  string    `gorm:"size:64;uniqueIndex;not null"` // 幂等键
	InstanceID     uint      `gorm:"index:idx_cmd_instance_status,priority:1;not null"`
	StrategyRunID  *uint     `gorm:"index"`
	Kind           string    `gorm:"size:32;index;not null"` // new_order|cancel|flatten
	Status         string    `gorm:"size:32;index:idx_cmd_instance_status,priority:2;not null;default:pending"` // pending|dispatched|acked|failed|dead
	PayloadJSON    string    `gorm:"type:text;not null"`
	ErrorMessage   string    `gorm:"size:512"`
	DispatchedAt   *time.Time `gorm:"index"`
	AckedAt        *time.Time
	CreatedAt      time.Time  `gorm:"index"`
	UpdatedAt      time.Time
}
