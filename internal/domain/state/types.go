// Package state 定义全系统、执行面与行情面的聚合快照类型；供调度器、回测与状态收敛只读消费。
package state

import (
	"github.com/altshort/quant/internal/domain/instance"
	"github.com/altshort/quant/internal/domain/report"
)

// MarketStateSnapshot 为单一标的上的市场公开信息收敛（价格、资金费率等）。
// 真源为交易所行情；SaaS 与 Agent 可各自维护缓存副本，以交易所时间戳对齐。
type MarketStateSnapshot struct {
	Symbol string `json:"symbol"`

	LastPrice float64 `json:"last_price,omitempty"`
	BestBid   float64 `json:"best_bid,omitempty"`
	BestAsk   float64 `json:"best_ask,omitempty"`

	// MarkPrice 为合约标记价（现货策略可为 0）。
	MarkPrice float64 `json:"mark_price,omitempty"`

	// FundingRate 为当前或下一期资金费率快照（无量纲小数）。
	FundingRate float64 `json:"funding_rate,omitempty"`

	// ExchangeTimeUnixMs 为行情源时间（Unix 毫秒）。
	ExchangeTimeUnixMs int64 `json:"exchange_time_unix_ms"`
}

// ExecutionStateSnapshot 为 **单实例** 执行态折叠：持仓、挂单、账户摘要。
// 数据通常源自 Agent DeltaReport 链路的最近已知状态（控制面缓存）；交易所以 WS/REST 为最终事实来源，经 Agent 归一化。
type ExecutionStateSnapshot struct {
	InstanceID string `json:"instance_id"`

	OpenOrders []report.OpenOrderSnapshot `json:"open_orders,omitempty"`
	Positions  []report.PositionSnapshot  `json:"positions,omitempty"`
	Account    *report.AccountSnapshot   `json:"account,omitempty"`

	// LastDeltaReportID 为折叠所使用的最近增量包 id（便于诊断）。
	LastDeltaReportID string `json:"last_delta_report_id,omitempty"`
}

// SystemStateSnapshot 描述某一时刻整个系统的可观测切面：实例编排状态、各市场行情、各实例执行折叠。
// 用于回放对齐、监控与回测引擎的外部状态注入接口；不包含策略内部私有记忆。
type SystemStateSnapshot struct {
	// CapturedAtUnixMs 为记录该快照的逻辑时间（Unix 毫秒），由写入方注入。
	CapturedAtUnixMs int64 `json:"captured_at_unix_ms"`

	// Instances 为控制面已知的实例状态列表。
	Instances []instance.InstanceState `json:"instances,omitempty"`

	// Markets 按 symbol 索引的最近市场快照（缓存或实时；策略 Step 的 Bar/Features 仍由上游特征管线拼装）。
	Markets map[string]MarketStateSnapshot `json:"markets,omitempty"`

	// ExecutionByInstance 按 InstanceID 索引的执行折叠视图。
	ExecutionByInstance map[string]ExecutionStateSnapshot `json:"execution_by_instance,omitempty"`
}
