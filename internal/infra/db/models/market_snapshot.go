package models

import (
	"time"
)

// MarketSnapshot 某时刻市场状态的持久化切片（K 线、盘口摘要等），供分析与审计。
// 真源：SaaS 表内记录为控制面侧「已采纳并落库」的快照（数据可能来自 Agent 上报或外部行情接入）；非内存缓存真源。
// 软删除：不支持（时序事实 append-oriented；纠错用新快照或补偿记录）。
type MarketSnapshot struct {
	ID          uint      `gorm:"primaryKey"`
	Symbol      string    `gorm:"size:32;index:idx_snapshot_symbol_captured,priority:1;not null"`
	CapturedAt  time.Time `gorm:"index:idx_snapshot_symbol_captured,priority:2;not null"`
	Source      string    `gorm:"size:64;index;not null"` // e.g. venue_ws, agent_push, saas_feed
	InstanceID  *uint     `gorm:"index"`
	StrategyRunID *uint   `gorm:"index"`
	PayloadJSON string    `gorm:"type:text;not null"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}
