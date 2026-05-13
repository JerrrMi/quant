// Package strategy 定义策略纯函数的输入/输出契约与相关快照类型；不含实现与 I/O。
package strategy

import "github.com/JerrrMi/quant/internal/domain"

// RiskCompositeNormalizedKey 为 MinimalShortStep 识别的综合风险占位键；特征管线填入 [0,1]，缺失视为低风险 0。
// 完整策略可改用自有键集合，同时在文档标注迁移。
const RiskCompositeNormalizedKey = "risk_composite_01"

// MarketFeatureSnapshot 是单步可用的市场特征切片：同时可含（a）当前点上的归一化/无量纲量、（b）覆盖固定回溯窗口的统计量摘要。
// 全部为只读输入：由上游特征管线或回测引擎在调用 Step 之前拼装；Step 内不得修改。
type MarketFeatureSnapshot struct {
	// Normalized 为_namespaced_ 的归一化或无量纲标量，取值区间由键名契约约定（例如滚动 Z-score、分位数映射到 [-1,1]）。
	// 语义：当前快照时刻的点值，而非整个窗口序列。
	Normalized map[string]float64 `json:"normalized,omitempty"`

	// WindowStats 为基于固定窗口长度的统计摘要（例如近 N 根的波动率、成交量分位等）；键名需标明窗口长度约定。
	// 语义：窗口统计量，对应窗口右端为当前 Bar 或当前 tick（由上游约定）。
	WindowStats map[string]float64 `json:"window_stats,omitempty"`

	// RawTags 为非数值的离散标签（如体制/regime 分类），仅供策略只读分支使用。
	RawTags map[string]string `json:"raw_tags,omitempty"`
}

// RiskSnapshot 描述本步决策时可用的风险与约束视图（只读输入）。
// 来源：SaaS 配置与账户/持仓真源经 Agent 汇聚后的策略可见子集；具体字段由风控模块填充。
type RiskSnapshot struct {
	// MaxLeverage 允许的最大杠杆倍数（快照值，非实时拉取）。
	MaxLeverage float64 `json:"max_leverage,omitempty"`

	// MaxNotionalUSDT 当前策略实例在该标的或账户维度上的名义额度上限（计价约定为 USDT 等价快照）。
	MaxNotionalUSDT float64 `json:"max_notional_usdt,omitempty"`

	// CurrentDrawdown01 为归一化到 [0,1] 的回撤强度或预算消耗比例（具体定义由风控文档约定）。
	CurrentDrawdown01 float64 `json:"current_drawdown_01,omitempty"`

	// TradingHalted 为 true 时策略应不产生开新仓意图（只读标志）。
	TradingHalted bool `json:"trading_halted,omitempty"`
}

// MinimalSkeletonParams 为最小做空骨架的可选参数，由编排层或回测注入；全零字段表示使用包内默认常量。
// 完整策略落地后可忽略本字段或迁移为独立版本化参数结构体。
type MinimalSkeletonParams struct {
	// MaxShortHoldMs 空仓自 ShortOpenedAtUnixMs 起最长持有毫秒，超时则平仓。
	MaxShortHoldMs int64 `json:"max_short_hold_ms,omitempty"`
	// EntryLogReturn 为开仓所需的非正对数收益阈值（ ln(close/prior) 需 ≤ 该值，通常为负小数）。
	EntryLogReturn float64 `json:"entry_log_return,omitempty"`
	// TakeProfitLogReturn 为止盈：空仓时 ln(close/prior) ≥ 该正值则平空。
	TakeProfitLogReturn float64 `json:"take_profit_log_return,omitempty"`
	// StopLossLogReturn 为止损：空仓时 ln(close/prior) ≥ 该正值（价格反弹）则平空。
	StopLossLogReturn float64 `json:"stop_loss_log_return,omitempty"`
	// ReduceLogReturnMin 为减仓区间下界：空仓时收益在该下界与 TakeProfit 之间则减仓。
	ReduceLogReturnMin float64 `json:"reduce_log_return_min,omitempty"`
}

// AltShortStrategyInput 是单步策略计算的单次输入：只含快照与只读字段，供纯函数 Step 使用。
type AltShortStrategyInput struct {
	// Symbol 为交易对标识（venue 规范化，例如 "BTCUSDT"）；只读输入。
	Symbol string `json:"symbol"`

	// NetPositionQty 为该标的策略可见净持仓：多仓为正、空仓为负；绝对值接近 0 视为无仓。
	NetPositionQty float64 `json:"net_position_qty"`

	// ShortOpenedAtUnixMs 为当前空仓的开仓逻辑时间（毫秒，与 NowUnixMs 同源）；非空仓时应为 0。
	ShortOpenedAtUnixMs int64 `json:"short_opened_at_unix_ms,omitempty"`

	// PriorBarClose 为上一根已完结 K 线的收盘价（与 BarCurrent 同标的），用于简易对数收益；不可得时为 0。
	PriorBarClose float64 `json:"prior_bar_close,omitempty"`

	// BarCurrent 为当前步对应的已完结 K 线快照（原始量纲）；只读输入。
	BarCurrent domain.Bar `json:"bar_current"`

	// Features 为当前步的市场特征切片（归一化值与窗口统计量等）；只读输入。
	Features MarketFeatureSnapshot `json:"features"`

	// Risk 为当前步可见的风险/约束快照；只读输入。
	Risk RiskSnapshot `json:"risk"`

	// NowUnixMs 为本次 Step 的逻辑时间（Unix 毫秒），由调度器或回测时钟注入；策略内不得读取系统墙钟。
	NowUnixMs int64 `json:"now_unix_ms"`

	// StepSequence 为实例内单调递增的步序号；只读输入，用于日志与诊断对齐。
	StepSequence int64 `json:"step_sequence"`

	// Minimal 为最小骨架专用参数钩子；可为 nil。
	Minimal *MinimalSkeletonParams `json:"minimal,omitempty"`
}

// StrategySignal 是策略在本步的离散/连续综合输出，供下游编排或人工审计（策略输出结果）。
type StrategySignal struct {
	// Name 为信号名称（例如 "short_entry", "flatten"）。
	Name string `json:"name"`

	// Strength 为 [-1,1] 或可文档化的连续强度，表示相对置信或目标尺度；无量纲。
	Strength float64 `json:"strength"`

	// ReasonDetail 为人类可读的一句决策摘要（审计）；区别于 ReasonCodes 的短码枚举。
	ReasonDetail string `json:"reason_detail,omitempty"`

	// ReasonCodes 为可机器处理的简短原因码列表（策略输出）。
	ReasonCodes []string `json:"reason_codes,omitempty"`

	// ValidUntilUnixMs 为本步信号的参考失效时间（毫秒，与注入时钟同源）；0 表示未设置。
	ValidUntilUnixMs int64 `json:"valid_until_unix_ms,omitempty"`

	// Confidence01 为本步离散决策的置信度 [0,1]；可与 Strength 解耦时使用。
	Confidence01 float64 `json:"confidence_01,omitempty"`
}

// TradeIntent 描述策略希望在执行层被翻译为命令的一条意图（策略输出结果，非交易所状态）。
type TradeIntent struct {
	// IntentID 在单次 AltShortStrategyOutput 内唯一，用于与 Command 关联。
	IntentID string `json:"intent_id"`

	// Symbol 冗余携带标的，便于多标的扩展；应与输入 Symbol 一致除非策略明确做跨标的意图。
	Symbol string `json:"symbol"`

	// Side 为执行侧解释的买卖方向；与领域 domain.Side 一致。
	Side domain.Side `json:"side"`

	// TargetNotionalUSDT 为意图的名义目标（USDT 计价快照）；与 TargetPosition 二选一由 IsReduceOnly/上游约定标明优先级。
	TargetNotionalUSDT *float64 `json:"target_notional_usdt,omitempty"`

	// TargetPositionQty 为标的数量维度的目标净持仓（基币数量；正号约定由执行层与白名单文档统一）。
	TargetPositionQty *float64 `json:"target_position_qty,omitempty"`

	// IsReduceOnly 为 true 时仅允许减仓（策略输出标志）。
	IsReduceOnly bool `json:"is_reduce_only,omitempty"`

	// Urgency01 为 [0,1] 归一化紧迫度，供调度/拆单参考（策略输出）。
	Urgency01 float64 `json:"urgency_01,omitempty"`
}

// AltShortStrategyOutput 为单步策略计算的输出：信号与交易意图列表等均为策略侧结果。
type AltShortStrategyOutput struct {
	Signal StrategySignal `json:"signal"`

	// Intents 可包含零条或多条；空表示本步不下达执行意图。
	Intents []TradeIntent `json:"intents,omitempty"`

	// Diagnostics 为可选的额外归一化标量（策略输出；仅供记录与回测对齐）。
	Diagnostics map[string]float64 `json:"diagnostics,omitempty"`
}
