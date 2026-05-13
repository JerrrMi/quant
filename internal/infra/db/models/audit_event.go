package models

import (
	"time"
)

// AuditEvent 控制面动作与关键变更的不可变审计日志（谁对什么资源做了什么）。
// 真源：SaaS（Agent 不写本表或由同步服务写入只读副本；默认仅 SaaS 写入）。
// 软删除：不支持。
type AuditEvent struct {
	ID           uint      `gorm:"primaryKey"`
	ActorType    string    `gorm:"size:32;index;not null"` // user|system|api_token
	ActorID      string    `gorm:"size:128;index"`
	Action       string    `gorm:"size:64;index;not null"`
	ResourceType string    `gorm:"size:64;index:idx_audit_resource,priority:1"`
	ResourceID   string    `gorm:"size:128;index:idx_audit_resource,priority:2"`
	PayloadJSON  string    `gorm:"type:text"`
	OccurredAt   time.Time `gorm:"index;not null"`
	CreatedAt    time.Time `gorm:"index"`
	UpdatedAt    time.Time // 与 GORM 约定一致；审计行创建后不应更新
}
