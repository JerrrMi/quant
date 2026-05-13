// Package domain 定义领域类型与策略纯函数契约；不含 I/O 与第三方 import。
package domain

// Bar 是单根 K 线占位（无量纲化前的原始 OHLCVT 可映射到此处）。
type Bar struct {
	Open, High, Low, Close float64
	Volume                 float64
	// TimestampUnixMs 使用规范化的 Unix 毫秒，避免策略直接读墙钟；由上游注入。
	TimestampUnixMs int64
}

// StepInput 聚合进入策略 Step 的归一化输入；字段随策略 Phase 扩展。
type StepInput struct {
	Symbol string // 规范化符号，如 "BTCUSDT"
	Bar    Bar
}

// StepOutput 表示策略单步输出（信号/目标仓位等）；占位。
type StepOutput struct {
	// TargetExposure 示例：[-1,1] 归一化目标敞口；具体语义在策略文档中约定。
	TargetExposure float64
}

// Strategy 是纯策略函数接口占位；实现应为无外部依赖的纯函数或闭包。
// 调用方：回测引擎与实盘调度在构造 StepInput 后调用。
type Strategy interface {
	Step(in StepInput) (StepOutput, error)
}
