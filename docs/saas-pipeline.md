# SaaS 数据管线与调度链

本文描述控制面（`cmd/saas`）侧：**市场数据如何进入系统**、**LPPL 等外源特征如何进入 `AltShortStrategyInput`**、**`Step()` 如何被触发**、**策略输出如何变为 `TradeCommand`**、**命令如何落库并通过 WebSocket 下发到 Agent**，以及**幂等边界**。

实现锚点：

- 市场数据与特征辅助：`internal/infra/marketdata`
- LPPL 扫描承接与输入适配：`internal/infra/lppl`
- 定时编排（无择时公式）：`internal/scheduler`（`StepOrchestrator`、`CronTrigger`）
- 进程生命周期与装配：`internal/app/saas`
- Agent WebSocket Hub：`internal/infra/ws`（`AgentHub`、`SaasAgentServer`）
- 域类型：`internal/domain/strategy`（`AltShortStrategyInput` / `AltShortStrategyOutput`）、`internal/domain/command`（`TradeCommand`）

**约束回顾**：SaaS **不**直连 Binance（或任意交易所）；**不**在调度器内写策略/风控业务裁决；策略核心仍仅为纯函数 `Step()`。

---

## 1. 市场数据如何进入系统

1. **真源边界**：实盘控制面上的 `MarketSnapshot` 表示「已被 SaaS 采纳并落库」的行情切片，来源可以是 Agent 增量上报、内部聚合服务或其他 **非 SaaS 直连交易所** 的上游（具体接入在 infra 层替换实现）。
2. **读取路径**：`internal/infra/marketdata.DBBarSeriesReader` 从 `MarketSnapshot` 表按 `symbol` 逆时取最近 `N` 条，将 `PayloadJSON` 解码为 `SnapshotPayload`（内含 `domain.Bar`），再按时间正序返回，供窗口与特征构造使用。
3. **特征辅助**：`BuildPriceWindow`、`LogReturnLast`、`LogReturnsFromCloses`、`WindowZScoreLast`、`BuildFeatureSnapshot` 等在 **无量纲/归一化** 空间组装 `strategy.MarketFeatureSnapshot`，写入 `AltShortStrategyInput.Features`，键名为编排契约（非策略公式）。
4. **数据刷新循环**：`internal/app/saas.Runner.runDataLoop` 按 `configs/saas.yaml` 中 `data_pipeline.refresh_interval_seconds` 周期唤醒（当前为占位日志；生产应替换为「从 Agent/聚合器拉取并 `INSERT` 快照」的实现）。

---

## 2. LPPL 如何作为外部特征进入 `AltShortStrategyInput`

1. **存储**：LPPL（或同类）离线/准实时作业将结果写入 `LPPLScanResult`（`JobID` 用于与外部任务对齐；`ResultJSON` 为 JSON 载荷）。
2. **读取端口**：`internal/infra/lppl.GormResultStore` 实现 `ResultStore.Save` / `LatestBySymbol`。
3. **契约**：`ResultScalars`（如 `bubble_metric_01`）为建议字段；`ParseScalars` 宽松解码，缺失不令整链失败。
4. **注入**：`lppl.InputAugmentor.ApplyLatest` 将最新结果的标量写入 `in.Features.Normalized["lppl_bubble_metric_01"]`，标签写入 `RawTags`（如 `lppl_job_id`）。**仅填充特征**，不改变 `NetPositionQty` / 风险等账户语义字段。

---

## 3. Step 如何被调用

1. **触发**：`scheduler.CronTrigger` 在 `scheduler.cron_expression` 非空时优先使用 **cron**（`github.com/robfig/cron/v3`），否则使用 `tick_interval_seconds` 的 **固定 ticker**（配置见 `internal/config.SaaSConfig`）。
2. **编排入口**：每次触发调用 `StepOrchestrator.Tick`：列出 `status=active` 的 `Instance`，对每个实例 `EnsureRunningRun` 得到 `StrategyRun`，计算 `nextSeq = LastStepSequence+1`。
3. **输入组装**：从 `DBBarSeriesReader` 拉取窗口 → `BuildFeatureSnapshot` → `InputAugmentor.ApplyLatest`（LPPL）→ 填入 `AltShortStrategyInput`（`NowUnixMs`、`StepSequence` 由编排注入）。
4. **可追溯性**：在调用 `Stepper.Step` 前写入 `AuditEvent`（`strategy.step_input`，载荷为完整输入 JSON）；`Step` 后写入 `strategy.step_output`。若行情不足，写 `strategy.step_skipped` 并跳过 `Step`。
5. **纯策略调用**：`Stepper` 默认为 `strategy.MinimalShortStrategy`，仅做纯函数计算；调度器 **不包含** 择时规则。

---

## 4. Step 输出如何转成命令

1. `AltShortStrategyOutput.Intents` 中的每条 `TradeIntent` 由编排映射为一条 `command.TradeCommand`（`Kind` 暂为 `place`；`CommandID` / `Nonce` 由 SaaS 分配 UUID；`IdempotencyKey` 见下节）。
2. `TradeCommand` 序列化入 `TradeCommandRecord.PayloadJSON`，表字段 `CorrelationID = string(IdempotencyKey)`。

---

## 5. 命令如何落库并下发到 Agent

1. **落库**：`repository.GormCommandRepository.SaveCommand` 写入 `TradeCommandRecord`，初始 `Status=pending`；审计 `command.persisted`。
2. **WebSocket**：`internal/infra/ws.SaasAgentServer` 接受 Agent 连接；首帧 `auth` 的 `client_id` 必须等于实例的 `AgentKey`，并注册到 `AgentHub`。
3. **下发**：`saas.HubDispatcher` 实现 `scheduler.CommandDispatcher`，调用 `AgentHub.SendCommand` → `Peer.Send`，`type=command`，载荷为 `TradeCommand`。成功后更新 `Status=dispatched` 与 `DispatchedAt`，并写审计 `command.dispatched`。
4. **无连接**：若 Agent 未在线，`Dispatch` 失败，记录保持 `pending`，依赖后续重试路径（重放可由同一 `IdempotencyKey` 驱动）。

---

## 6. 幂等与必须保持幂等的环节

| 环节 | 幂等键 / 行为 |
|------|----------------|
| **交易指令** | `TradeCommandRecord.CorrelationID`（由 `inst/run/step/intent` 组成）；已存在且已 `dispatched` 则跳过重发；`pending` 时重用已存 JSON 载荷再试发。 |
| **LPPL 结果** | `LPPLScanResult.JobID` 建议唯一；重复作业应由上游处理「覆盖或跳过」。 |
| **策略 Step 审计** | 每次 tick 追加新审计事件；以 `strategy_run` + `StepSequence` 对齐，不作为「仅一次」去重键。 |
| **Cron / 重入** | 同一逻辑时间多实例进程应靠 DB 唯一约束 + 相关键避免重复下单；Agent 侧仍以 `IdempotencyKey` 为最终去重。 |

---

## 7. SaaS 生命周期接口

`internal/app/saas` 提供：

- `Run(ctx, cfg, deps)`：启动 HTTP/WebSocket、可选调度与数据循环，阻塞至 `ctx` 取消后 `Shutdown`。
- `Runner.Start` / `Runner.Stop`：与同进程内显式启停配合使用（与 `Run` 二选一约定）。

---

## 8. 状态恢复

进程启动时 `Runner.recoverState` 统计 `running` 的 `StrategyRun` 数量并写 `AuditEvent`（`saas.state_recovery`）。**Step 序号**持久于 `StrategyRun.LastStepSequence`（GORM `AutoMigrate` 增加列），调度器恢复后从该列继续递增。
