package strategy

import "github.com/JerrrMi/quant/internal/domain"

// Stepper 是纯策略单步计算契约：给定完整输入快照，返回输出；不得产生 I/O、不得依赖执行侧。
type Stepper interface {
	Step(input AltShortStrategyInput) (AltShortStrategyOutput, error)
}

// StrategyDataProvider 供编排层拉取只读数据以拼装 AltShortStrategyInput；
// 禁止包含下单、撤单或任何写交易所状态的能力；实现可位于 infra/backtest，不在此包。
type StrategyDataProvider interface {
	// LatestClosedBar 返回标的最新一根已完结 K 线（原始量纲快照）。
	LatestClosedBar(symbol string) (domain.Bar, error)
}
