package models

import (
	"time"
)

// WSSession 观测到的 WebSocket 会话（SaaS 监听侧或 Agent 出站连接侧均可落库；不以内存连接为真源）。
// 真源：进程本地 DB 中本行为该进程视角下的会话真源（SaaS 记接入、Agent 记出站）；跨进程一致事件以审计表/命令流为准。
// 软删除：不支持（用 ClosedAt 表达关闭；记录保留排障）。
type WSSession struct {
	ID           uint       `gorm:"primaryKey"`
	Scope        string     `gorm:"size:32;index;not null"` // saas_inbound|agent_outbound
	InstanceID   *uint      `gorm:"index"`
	AgentKey     string     `gorm:"size:128;index"`
	SessionKey   string     `gorm:"size:64;uniqueIndex;not null"` // 连接实例 id
	RemoteAddr   string     `gorm:"size:256"`
	ConnectedAt  time.Time  `gorm:"index;not null"`
	LastSeenAt   *time.Time `gorm:"index"`
	ClosedAt     *time.Time `gorm:"index"`
	CloseReason  string     `gorm:"size:256"`
	CreatedAt    time.Time  `gorm:"index"`
	UpdatedAt    time.Time
}
