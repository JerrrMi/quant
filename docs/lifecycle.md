# AltShort 进程生命周期

本文描述 **SaaS**、**Agent**、**Backtest** 三进程的启动顺序、WebSocket 重连策略、优雅停机与状态收敛约定。实现入口在 `internal/lifecycle`（管理器、重连策略、停机编排）；业务编排仍在 `internal/app` 与各 `cmd/*`，策略逻辑不得进入生命周期包。

## 启动顺序

### SaaS（`cmd/saas` → `internal/app/saas.Runner`）

由 `lifecycle.Manager` 按注册顺序拉起子系统（仅组织启动，不含领域决策）：

1. **recover_state**：加载/审计运行中策略 run 数量（`saas.state_recovery`）。
2. **http_websocket**：`ListenAndServe`，挂载 Agent WebSocket。
3. **scheduler_trigger**：若配置启用，启动 `CronTrigger`（独立 `schedCtx`，可与根 ctx 分别收敛）。
4. **data_pipeline_loop**：数据管线占位 ticker（独立 `dataCtx`）。

配置加载、日志、`BootstrapSaaS` 仍在 `cmd/saas` / `internal/app`，先于上述阶段完成。

### Agent（`cmd/agent` → `internal/app/agent.Run`）

1. 读取配置与环境变量中的交易所凭证（仅 Agent 进程）。
2. 装配执行器与本地 `DedupStore`。
3. 进入 `lifecycle.RunReconnectLoop`：会话内完成 WS 拨号、认证、心跳与命令循环。

### Backtest（`cmd/backtest`）

1. 加载回测 YAML、`BootstrapBacktest`。
2. `Engine.Run` 在每条 bar 前检查 `ctx`（支持 SIGTERM 协作退出）。
3. 无长连接；停机主要是进程信号与上下文取消。

## 重连策略（Agent ↔ SaaS）

实现：`internal/lifecycle/reconnect.go`，由 Agent 调用。

- **指数退避**：失败会话后休眠 `initial_backoff × multiplier^n`，默认乘数为 2。
- **最大间隔**：由 `reconnect.max_backoff_seconds` 封顶（含抖动后截断）。
- **抖动**：`reconnect.jitter_ratio`，避免多 Agent 同时重连。
- **会话次数上限**：`reconnect.max_attempts`，0 表示不限制。
- **认证成功后**：`ExpBackoff.Reset()`，下一次断线从初始退避重新算起；随后按配置的 **HeartbeatIntervalSecs** 启动心跳（“恢复正常心跳”）。
- **断线记录**：`ReconnectHooks.OnDisconnected` / `BeforeReconnect` 记录原因与时间戳（审计扩展可在 hooks 中挂载）。
- **重连后状态同步入口**：`ReconnectHooks.AfterAuthSuccess(ctx)`（可选）。在此处做 **幂等** 的对账（例如从 SaaS 拉取未确认命令），并与执行层 `DedupStore`/指令 `IdempotencyKey` 配合，避免重复成交。

**避免重复执行**：传输层重连不应对同一 `command_id` 再次下单；依赖持久化命令行的关联 ID、Agent 侧去重存储及 SaaS 侧 `dispatched` 状态。

## 停机策略（SaaS）

实现：`internal/lifecycle/shutdown.go` 中的 `ShutdownCoordinator`，由 `saas.Runner.runGracefulShutdown` 编排。

收到 `SIGINT`/`SIGTERM` 后（根 `ctx` 取消），顺序为：

1. **stop_accepting_new_work**：取消 `schedCtx` / `dataCtx`，`CronTrigger.Stop()`，不再调度新 tick，数据占位循环退出。
2. **wait_scheduler_tick_idle**：`StepOrchestrator.WaitIdle`（带超时）；超时仅记告警，**不跳过**后续快照。
3. **flush_shutdown_snapshot**：写入审计事件 `saas.shutdown_snapshot`（运行中 run 计数等）。**必须在关闭 DB 之前执行**。
4. **shutdown_http_websocket**：`http.Server.Shutdown`，关闭监听与现有 WebSocket。
5. **close_cache_clients**：若 `Deps.Cache` 实现 `io.Closer`，则关闭。
6. **close_database**：关闭 GORM 底层 SQL 连接。

默认停机总时限：`lifecycle.DefaultShutdownTimeout`（30s）。日志由 `ShutdownCoordinator` 逐步打出 **step begin / complete**。

Agent / Backtest：Agent 在根 ctx 取消时结束会话循环并重连逻辑退出；Backtest 在 bar 边界检测 ctx 取消并返回错误终止。

## 必须先落盘的关键状态

| 状态 | 说明 |
|------|------|
| 策略 run / step 序列 | `LastStepSequence`、run 状态机，避免重启重复 step。 |
| 指令行与幂等键 | `trade_commands`、关联 ID、`dispatched` 时间；重复派发由存储层短路。 |
| 审计事件 | 恢复与停机快照（`saas.state_recovery`、`saas.shutdown_snapshot`）。 |
| Agent 去重/执行游标 | `DedupStore`、`LastSeenSaasSeq`（认证载荷），用于断线续传与幂等。 |

## 断线后自动收敛

1. **传输**：Agent 按退避重连；认证成功后重置退避并恢复心跳。
2. **序号**：Agent 维护 SaaS 信封单调序号，重连后认证携带 `LastSeenSaasSeq`，由控制面决定是否重放（协议演进点）。
3. **命令**：已持久化且已标记 `dispatched` 的意图不得再次下发；重连后的 **`AfterAuthSuccess`** 应对悬挂命令做对账，且不绕过执行层去重。
4. **SaaS 调度**：停机时先停触发器再 `WaitIdle`，尽量让当前 `Tick` 写完仓储后再刷快照关库。

## 相关代码路径

- `internal/lifecycle/manager.go` — 组件注册与启停顺序、依赖挂载、观测快照。
- `internal/lifecycle/reconnect.go` — 退避、`RunReconnectLoop`、断线记录与认证后钩子。
- `internal/lifecycle/shutdown.go` — 停机步骤运行器与 DB/Cache 关闭助手。
- `internal/app/saas/runner.go` — SaaS 启动编排与停机步骤组装。
- `internal/app/agent/run.go` — 重连循环与认证后钩子接线。
- `internal/scheduler/trigger.go` — `CronTrigger.Stop()` 与 ticker/cron 收敛。
