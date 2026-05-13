// Package report 定义 Agent 向 SaaS 汇报的执行增量与账户/持仓快照结构；无 I/O。
package report

import "github.com/JerrrMi/quant/internal/domain"

// FillRecord 单条成交明细（交易所已成交事实；Agent 从 venue 拉取或推送解析后的真源子集）。
type FillRecord struct {
	// FillID 为交易所成交号或代理内稳定主键（Agent 归一化后填充）。
	FillID string `json:"fill_id"`

	// Symbol 标的。
	Symbol string `json:"symbol"`

	Side domain.Side `json:"side"`

	// Price 为成交价格（标的报价资产，通常为 USDT 对）。
	Price float64 `json:"price"`

	// Quantity 为成交数量（基币）。
	Quantity float64 `json:"quantity"`

	// Fee 为该笔成交费用（数值；计价资产见 FeeAsset）。
	Fee float64 `json:"fee"`

	// FeeAsset 为费用计价资产代码。
	FeeAsset string `json:"fee_asset,omitempty"`

	// ExchangeTradeTimeUnixMs 为交易所报告的成交时间（Unix 毫秒）。
	ExchangeTradeTimeUnixMs int64 `json:"exchange_trade_time_unix_ms"`

	// ClientOrderID 为关联的客户端订单标识（若有）。
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// OrderStatus 粗略描述未完结挂单状态（快照；与 venue 枚举可能粗对齐）。
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCanceled        OrderStatus = "canceled"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusExpired         OrderStatus = "expired"
)

// OpenOrderSnapshot 为尚未完全终结的挂单快照（Agent 从交易所 REST/WS 收敛）。
type OpenOrderSnapshot struct {
	ExchangeOrderID string      `json:"exchange_order_id"`
	Symbol          string      `json:"symbol"`
	Side            domain.Side `json:"side"`
	// Price 为限价；市价单语义由 Type 字段（若扩展）或价格为 0 约定。
	Price      float64     `json:"price"`
	Quantity   float64     `json:"quantity"`
	FilledQty  float64     `json:"filled_qty"`
	Status     OrderStatus `json:"status"`
	ReduceOnly bool        `json:"reduce_only,omitempty"`
	// ExchangeUpdateTimeUnixMs 为交易所订单状态最后更新时间（Unix 毫秒）。
	ExchangeUpdateTimeUnixMs int64 `json:"exchange_update_time_unix_ms"`
}

// PositionSnapshot 为单标的净持仓视图（衍生品语境下含方向与保证金占用等占位字段）。
type PositionSnapshot struct {
	Symbol string `json:"symbol"`

	// PositionQty 为净持仓数量（符号约定：多正空负或全正配合 Side 字段——由执行层文档统一；此处推荐带符号净仓）。
	PositionQty float64 `json:"position_qty"`

	// EntryPrice 为平均开仓价（若可得）。
	EntryPrice float64 `json:"entry_price,omitempty"`

	// UnrealizedPnlUSDT 为未实现盈亏（USDT 计价快照）。
	UnrealizedPnlUSDT float64 `json:"unrealized_pnl_usdt,omitempty"`

	// InitialMarginUSDT 为该仓位占用的初始保证金（快照）。
	InitialMarginUSDT float64 `json:"initial_margin_usdt,omitempty"`

	// MaintenanceMarginUSDT 为维护保证金（快照）。
	MaintenanceMarginUSDT float64 `json:"maintenance_margin_usdt,omitempty"`

	// Leverage 为当前杠杆倍数快照（若为逐仓/全仓由账户层补充）。
	Leverage float64 `json:"leverage,omitempty"`

	// ExchangePositionTimeUnixMs 为交易所仓位状态时间戳（Unix 毫秒）；不可得时为 0。
	ExchangePositionTimeUnixMs int64 `json:"exchange_position_time_unix_ms"`
}

// AccountSnapshot 为账户级资金与风险摘要（交易所账户真源经 Agent 收敛）。
type AccountSnapshot struct {
	// EquityUSDT 为总权益（USDT 计价快照）。
	EquityUSDT float64 `json:"equity_usdt"`

	// WalletBalanceUSDT 为钱包余额（不含未实现盈亏时由 venue 定义；字段语义以实现为准并在文档锁定）。
	WalletBalanceUSDT float64 `json:"wallet_balance_usdt,omitempty"`

	// AvailableBalanceUSDT 为可用余额（可开新仓）。
	AvailableBalanceUSDT float64 `json:"available_balance_usdt,omitempty"`

	// UsedMarginUSDT 为已用保证金。
	UsedMarginUSDT float64 `json:"used_margin_usdt,omitempty"`

	// UnrealizedPnlUSDT 为账户级未实现盈亏合计（若有）。
	UnrealizedPnlUSDT float64 `json:"unrealized_pnl_usdt,omitempty"`

	// ExchangeAccountTimeUnixMs 为交易所账户更新时间（Unix 毫秒）。
	ExchangeAccountTimeUnixMs int64 `json:"exchange_account_time_unix_ms"`
}

// DeltaReport 是 Agent 向控制面汇报的一次增量包：可包含成交、挂单、持仓与账户视图的刷新。
// 时间基准以交易所时间戳字段为准；接收侧可记录 ingested 时间（不在此结构体现，避免与 Agent 时钟混淆）。
type DeltaReport struct {
	// ReportID 为该增量包的唯一标识（Agent 或 SaaS 分配策略由管道约定；通常 Agent 生成 UUID）。
	ReportID string `json:"report_id"`

	// InstanceID 对应策略实例（SaaS 真源标识，Agent 透传）。
	InstanceID string `json:"instance_id"`

	// Fills 为本批次新增或重复的成交记录（重复由消费侧按 FillID 幂等）。
	Fills []FillRecord `json:"fills,omitempty"`

	// OpenOrders 为未成交挂单快照切片；可全量替换或增量由管道约定（默认可视为最近一次已知全量子集）。
	OpenOrders []OpenOrderSnapshot `json:"open_orders,omitempty"`

	// Positions 为各标的持仓快照；单标策略常仅含一条。
	Positions []PositionSnapshot `json:"positions,omitempty"`

	// Account 为账户级摘要（可选，按调度频率附带）。
	Account *AccountSnapshot `json:"account,omitempty"`

	// Errors 为 Agent 侧归一化错误消息（例如 REST 失败）；不等价于交易所拒单文本，详细文本可附在 Details。
	Errors []string `json:"errors,omitempty"`

	// Details 为结构化诊断键值（只读便于 SaaS 展示）。
	Details map[string]string `json:"details,omitempty"`

	// ExchangeEventTimeUnixMs 为该增量包主要事件对应的交易所时间（若无则 0）。
	ExchangeEventTimeUnixMs int64 `json:"exchange_event_time_unix_ms"`
}

// ReportAck 为 SaaS 对 DeltaReport 的接收确认（可选轻量契约）。
type ReportAck struct {
	ReportID string `json:"report_id"`
	Received bool   `json:"received"`

	// RefEnvelopeSeq 为被确认的 Agent → SaaS delta_report 信封 seq。
	RefEnvelopeSeq int64 `json:"ref_envelope_seq,omitempty"`

	Message          string `json:"message,omitempty"`
	ServerTimeUnixMs int64  `json:"server_time_unix_ms"`
}
