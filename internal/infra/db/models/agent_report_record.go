package models

import (
	"time"
)

// AgentReportRecord Agent 向控制面周期性或事件性上报（心跳、仓位、风险指标、执行摘要）。
// 真源：语义上「报告内容」由 Agent 产生；SaaS 库中本表为 **控制面接收并落库** 的审计真源（Agent 本地可有缓存副本，恢复以重上报 + SaaS 已存为准）。
// 软删除：不支持（报告流为追加型事实）。
type AgentReportRecord struct {
	ID            string    `gorm:"type:char(36);primaryKey"`
	InstanceID    uint      `gorm:"index:idx_report_instance_time,priority:1;not null"`
	StrategyRunID *uint     `gorm:"index"`
	ReportType    string    `gorm:"size:32;index;not null"` // heartbeat|position|risk|execution_summary
	AgentMsgID    string    `gorm:"size:64;index"`           // Agent 侧去重
	PayloadJSON   string    `gorm:"type:text;not null"`
	ReceivedAt    time.Time `gorm:"index:idx_report_instance_time,priority:2;not null"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time
}
