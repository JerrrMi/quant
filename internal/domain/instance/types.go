// Package instance 定义运行实例、策略与 Agent 等边界的状态枚举与快照；无 I/O。
package instance

// LifecycleState 描述实例在编排器眼中的粗粒度生命周期（控制面真源）。
type LifecycleState string

const (
	LifecycleStateCreated    LifecycleState = "created"
	LifecycleStateStarting   LifecycleState = "starting"
	LifecycleStateRunning    LifecycleState = "running"
	LifecycleStateDraining   LifecycleState = "draining"
	LifecycleStateStopped    LifecycleState = "stopped"
	LifecycleStateFailed     LifecycleState = "failed"
)

// StrategyState 描述策略 goroutine/回测迭代是否在被调度执行（与交易所连接无关）。
type StrategyState string

const (
	StrategyStateIdle     StrategyState = "idle"
	StrategyStateStepping StrategyState = "stepping"
	StrategyStatePaused   StrategyState = "paused"
	StrategyStateError    StrategyState = "error"
)

// AgentState 描述执行进程与交易所会话的健康度（Agent 真源，向 SaaS 汇报）。
type AgentState string

const (
	AgentStateDisconnected AgentState = "disconnected"
	AgentStateConnecting   AgentState = "connecting"
	AgentStateSyncing      AgentState = "syncing"
	AgentStateReady        AgentState = "ready"
	AgentStateDegraded     AgentState = "degraded"
	AgentStateHalted       AgentState = "halted"
)

// InstanceState 聚合单实例在控制面的可观测状态：把 **编排生命周期**、**策略调度**、**Agent 执行** 三者并列展示边界。
// 字段之间禁止混用语义（例如不得以 AgentState 表达策略暂停）。
type InstanceState struct {
	// InstanceID 由 SaaS 分配。
	InstanceID string `json:"instance_id"`

	// Lifecycle 为控制面主导的实例生命周期。
	Lifecycle LifecycleState `json:"lifecycle"`

	// Strategy 为策略 Step 调度状态（回测/实盘通用语义）。
	Strategy StrategyState `json:"strategy"`

	// Agent 为执行侧连接与会话状态（仅实盘；回测可固定为缺省）。
	Agent AgentState `json:"agent"`

	// LastTransitionUnixMs 为最近一次状态迁移的观测时间（控制面墙钟或逻辑时钟；文档约定）。
	LastTransitionUnixMs int64 `json:"last_transition_unix_ms"`

	// Notes 为人类可读辅助说明（非强类型）。
	Notes string `json:"notes,omitempty"`
}
