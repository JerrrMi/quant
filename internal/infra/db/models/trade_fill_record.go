package models

import (
	"time"
)

// TradeFillRecord 交易所成交/填充事件（逐笔或聚合）。
// 真源：Agent（唯一与 venue 对账的一方；SaaS 通过 AgentReportRecord 或同步任务获得副本，不以 SaaS 自建成交为权威）。
// 软删除：不支持（账务/回放要求不可变）。
type TradeFillRecord struct {
	ID            string    `gorm:"type:char(36);primaryKey"`
	CommandID     string    `gorm:"type:char(36);index;not null"`
	InstanceID    uint      `gorm:"index:idx_fill_instance_time,priority:1;not null"`
	StrategyRunID *uint     `gorm:"index"`
	Venue         string    `gorm:"size:32;uniqueIndex:idx_fill_venue_trade,priority:1;not null"`
	VenueTradeID  string    `gorm:"size:128;uniqueIndex:idx_fill_venue_trade,priority:2;not null"`
	VenueOrderID  string    `gorm:"size:128;index"`
	Symbol        string    `gorm:"size:32;index;not null"`
	Quantity      string    `gorm:"size:64;not null"`
	Price         string    `gorm:"size:64;not null"`
	Fee           string    `gorm:"size:64"`
	FeeAsset      string    `gorm:"size:32"`
	Liquidity     string    `gorm:"size:16"`
	FilledAt      time.Time `gorm:"index:idx_fill_instance_time,priority:2;not null"`
	RawJSON       string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time
}
