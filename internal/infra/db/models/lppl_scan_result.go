package models

import (
	"time"
)

// LPPLScanResult LPPL（或同类）扫描的离线/准实时分析结果。
// 真源：SaaS（分析任务提交与结果归属控制面；数值由离线作业写入，非 Agent 执行真值）。
// 软删除：不支持（分析结果版本以新行替代；保留历史对比）。
type LPPLScanResult struct {
	ID            uint      `gorm:"primaryKey"`
	Symbol        string    `gorm:"size:32;index;not null"`
	WindowStart   time.Time `gorm:"index"`
	WindowEnd     time.Time `gorm:"index"`
	ComputedAt    time.Time `gorm:"index"`
	JobID         string    `gorm:"size:64;index"` // 幂等/去重
	ParamsJSON    string    `gorm:"type:text"`
	ResultJSON    string    `gorm:"type:text;not null"`
	StrategyRunID *uint     `gorm:"index"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time
}
