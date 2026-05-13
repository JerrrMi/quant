# AltShort 策略契约（最小做空骨架）

## 定位

仓库当前实现的 `internal/domain/strategy` 是一套 **最小可运行做空骨架**：目标仅为 **连通调度 → `Step()` → `AltShortStrategyOutput` → 编排/执行** 的类型管线和少量确定性占位规则。**不是** 生产级择时模型，也不在本文档复述完整 Alpha 说明书。

canonical 入口：**包级函数** `strategy.Step`，内部当前委派至 `MinimalShortStep`（纯函数）。

## 未来完整策略如何替换

1. **保持输入/输出形态不变**：新策略仍应为 `AltShortStrategyInput -> AltShortStrategyOutput`（或保留 `Step` 外层包装，将实现体换成新版本纯函数）。
2. **将实现移入单独文件/包**：在 `internal/domain/strategy/` 增设 `advanced_foo_strategy.go` 等文件，并实现同一签名的纯函数。
3. **切换装配点**：在回测循环与 Agent/SaaS 编排处，将 `Step` 的指向从 `MinimalShortStep` 改为新函数；若使用 `Stepper` 接口，则替换具体实现类型。
4. **版本化参数**：生产策略建议将参数从 `MinimalSkeletonParams` 演进为独立、可版本化的配置结构，由上游注入 `AltShortStrategyInput` 的扩展字段（见下节「可扩展」）。

## 强约束与可扩展

| 类别 | 说明 |
|------|------|
| **强约束** | `Step`（及任何策略体）必须 **纯函数**：不得网络、数据库、墙钟、`rand`、文件 I/O、日志；时间仅使用输入中的 `NowUnixMs`；不得在此层调用执行器。 |
| **强约束** | 输入中的 `Symbol`、`BarCurrent`、`Features`、`Risk`、`NowUnixMs`、`StepSequence` 为调度/特征/风控管线的核心锚点；执行侧应能依赖 `TradeIntent.Symbol`/`Side`/目标字段的稳定语义。 |
| **强约束** | `strategy` 包仅依赖领域原语（如 `domain.Bar`）；不得依赖 `internal/infra` / `executor`。 |
| **可扩展** | `MarketFeatureSnapshot.Normalized` / `WindowStats` / `Diagnostics`：**键空间可演进**，须在文档中命名空间化并约定取值范围。 |
| **可扩展** | `StrategySignal.ReasonCodes` / `ReasonDetail`：**原因码可按策略版本扩展**。 |
| **骨架专用** | `MinimalSkeletonParams`、`ReasonCode*` 常量仅服务最小范例；可被完整策略的参数模型替代或忽略。 |

## 回测与实盘共用同一 `Step`

- 两者均应 **拼装同一形状的 `AltShortStrategyInput`**（同一字段语义、同一时间轴约定、同一持仓符号约定）。
- **`NowUnixMs` 与 `StepSequence` 必须由引擎注入**：回测用心跳/Bar 时钟，实盘由调度器带逻辑时间，禁止使用 `time.Now()`。
- **`NetPositionQty` / `ShortOpenedAtUnixMs` / `PriorBarClose`** 由上游状态机维护；策略只读取，不读写交易所。
- **`Features.Normalized[strategy.RiskCompositeNormalizedKey]`** 为骨架示例键（`[0,1]`）；真实管线可对齐常量或更名——迁移时同步更新契约文档与填充逻辑。

## 相关源码

| 路径 | 说明 |
|------|------|
| `internal/domain/strategy/types.go` | 输入输出与快照类型 |
| `internal/domain/strategy/step.go` | `Step()` 门面 |
| `internal/domain/strategy/minimal_short_strategy.go` | `MinimalShortStep` 占位实现 |
| `internal/domain/strategy/interfaces.go` | `Stepper` 等小接口 |
