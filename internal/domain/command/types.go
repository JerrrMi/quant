// Package command 定义自 SaaS/编排下发至 Agent 的交易指令值对象；无 I/O、无 GORM。
package command

import (
	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/strategy"
)

// IdempotencyKey 为幂等键：同一自然操作在重试时复用相同键，由调度器分配；Agent 据此去重或对应 venue clientOrderId。
type IdempotencyKey string

// CommandKind 区分指令形态（下单/撤单/改单等）；与具体协议映射由执行层完成。
type CommandKind string

const (
	// CommandKindPlace 表示新建订单类指令。
	CommandKindPlace CommandKind = "place"
	// CommandKindCancel 表示撤销指定交易所订单。
	CommandKindCancel CommandKind = "cancel"
	// CommandKindReplace 表示改价改量（若 venue 不支持则由执行层降级）。
	CommandKindReplace CommandKind = "replace"
)

// CommandStatus 描述指令在控制面与执行面的生命周期状态。
type CommandStatus string

const (
	CommandStatusPending    CommandStatus = "pending"
	CommandStatusAccepted   CommandStatus = "accepted"
	CommandStatusRejected   CommandStatus = "rejected"
	CommandStatusWorking    CommandStatus = "working"
	CommandStatusCompleted  CommandStatus = "completed"
	CommandStatusCanceled   CommandStatus = "canceled"
	CommandStatusExpired    CommandStatus = "expired"
	CommandStatusDeadletter CommandStatus = "deadletter"
)

// TradeCommand 是一条可审计的、发往 Agent 的交易意图实例化；字段以 JSON 序列化友好为主。
// SaaS 为 command_id / instance_id / strategy_id 等控制面真源的分配者；Agent 校验后加入执行队列。
type TradeCommand struct {
	// CommandID 全局唯一，由 SaaS 或上游编排生成。
	CommandID string `json:"command_id"`

	// InstanceID 将指令关联到运行的策略实例（SaaS 真源）。
	InstanceID string `json:"instance_id"`

	// StrategyID 逻辑策略定义标识，用于配额与路由（SaaS 真源）。
	StrategyID string `json:"strategy_id"`

	// Symbol 标的（venue 规范化）。
	Symbol string `json:"symbol"`

	// Side 执行侧解释的买卖方向。
	Side domain.Side `json:"side"`

	// Intent 为源自策略输出的意图副本或精炼子集；执行层以此为主要语义参考。
	Intent strategy.TradeIntent `json:"intent"`

	// TargetNotionalUSDT 为本次指令的名义目标（USDT 计价）；可与 Intent 内字段并存，以本字段为编排最终值（若有则覆盖 Intent 内同名语义）。
	TargetNotionalUSDT *float64 `json:"target_notional_usdt,omitempty"`

	// TargetPositionQty 为基币数量维度的目标净仓；与 TargetNotionalUSDT 二选一或按 Kind 解释优先级由执行文档约定。
	TargetPositionQty *float64 `json:"target_position_qty,omitempty"`

	// ReduceOnly 为 true 时强制减仓单语义。
	ReduceOnly bool `json:"reduce_only"`

	// DeadlineUnixMs 为指令失效的绝对时间（Unix 毫秒）；过期后 Agent 不得再向交易所提交。
	DeadlineUnixMs int64 `json:"deadline_unix_ms"`

	// Nonce 为发起方单次批次的随机串或单调值，用于调试与补充去重维度。
	Nonce string `json:"nonce"`

	// IdempotencyKey 为跨重试的稳定键。
	IdempotencyKey IdempotencyKey `json:"idempotency_key"`

	// Kind 为指令类型（下单/撤单/改单）。
	Kind CommandKind `json:"kind"`
}

// CommandAck 是 Agent 对单条 TradeCommand 的接收或校验回执（Agent 产生的短时真源，同步至 SaaS）。
type CommandAck struct {
	CommandID string        `json:"command_id"`
	Status    CommandStatus `json:"status"`
	// Message 为人类可读原因（拒绝/降级时填充）。
	Message string `json:"message,omitempty"`
	// ExchangeOrderID 在 Accepted/Working 后可填交易所订单号（Agent 真源）。
	ExchangeOrderID string `json:"exchange_order_id,omitempty"`
	// AgentTimeUnixMs 为 Agent 生成该回执时的逻辑时间（Unix 毫秒）。
	AgentTimeUnixMs int64 `json:"agent_time_unix_ms"`
}
