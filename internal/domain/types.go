// Package domain 定义跨子包共享的领域原语；无 I/O、无基础设施依赖。
// 策略 Step 契约见 internal/domain/strategy；按子域拆分的类型见 strategy/command/report 等。
package domain

// Bar 是单根 K 线快照（上游注入的原始 OHLCVT；非归一化、非窗口统计）。
type Bar struct {
	Open, High, Low, Close float64
	Volume                 float64
	// TimestampUnixMs 为该 Bar 所报时间的 Unix 毫秒；由加载器/回放引擎注入，策略内只作只读输入。
	TimestampUnixMs int64
}

// Side 表示在单一交易对上与下单方向相关的单边语义（只读枚举值；JSON 与交易所映射由执行层完成）。
type Side string

const (
	// SideBuy 表示买入基差方向（现货/合约语义由合约类型在元数据中解释）。
	SideBuy Side = "buy"
	// SideSell 表示卖出基差方向。
	SideSell Side = "sell"
)
