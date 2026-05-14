package models

import (
	"time"
)

// StrategyRun 一次可观测的策略运行周期（与调度/人工启停对齐）。
// 真源：SaaS（运行 ID 的权威分配与生命周期；Agent 通过回报推进状态，不以本地为真源覆盖控制面）。
// 软删除：不支持（用 Status + EndedAt 表达终结；保留完整运行史）。
type StrategyRun struct {
	ID          uint       `gorm:"primaryKey"`
	InstanceID  uint       `gorm:"index:idx_run_instance_status,priority:1;not null"`
	StrategyID  uint       `gorm:"index;not null"`
	Status      string     `gorm:"size:32;index:idx_run_instance_status,priority:2;not null;default:pending"` // pending|running|stopped|failed
	// LastStepSequence 为本条运行已持久化执行的 Step 序号（恢复调度与幂等键组成用）。
	LastStepSequence int64 `gorm:"not null;default:0"`
	StartedAt   *time.Time `gorm:"index"`
	EndedAt     *time.Time `gorm:"index"`
	Note        string     `gorm:"size:512"`
	CreatedAt   time.Time  `gorm:"index"`
	UpdatedAt   time.Time
}
