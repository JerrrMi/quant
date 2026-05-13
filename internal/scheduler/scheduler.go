// Package scheduler 负责何时评估策略与投递到执行层；不含密钥与策略公式。
package scheduler

import "time"

// Scheduler 协调 tick/定时器与策略评估节拍。
type Scheduler struct {
	// Interval 是两次评估之间的名义间隔；实盘可结合交易所推送再校准。
	Interval time.Duration
}

// NewScheduler 创建调度器；调用方：cmd/agent 或 app 在 Phase 中注入。
func NewScheduler(interval time.Duration) *Scheduler {
	return &Scheduler{Interval: interval}
}
