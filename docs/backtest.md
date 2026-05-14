# 回测与实盘同构说明

## 必须同构的部分

1. **策略入口**：回测与 SaaS 编排层都调用 **`strategy.Step(AltShortStrategyInput)`**（`internal/domain/strategy/step.go`），不允许在回测目录里复制第二套择时逻辑或平行实现。
2. **输入契约**：每一步的 **`AltShortStrategyInput`** 与实盘一致——同一组字段（`Symbol`、`NetPositionQty`、`ShortOpenedAtUnixMs`、`PriorBarClose`、`BarCurrent`、`Features`、`Risk`、`NowUnixMs`、`StepSequence`、`Minimal`）。特征构造复用 **`marketdata.BuildFeatureSnapshot`**，与 `internal/scheduler/orchestrator.go` 中的窗口长度约定一致（`signal_lookback` → `FeatureWindowSpec.WindowBars`）。
3. **LPPL**：可选融合路径与实盘相同——`**lppl.InputAugmentor.ApplyLatest**` 只写入 `Features`，不篡改持仓类字段。离线场景可使用 `lppl.FixedLatestStore` 注入一条静态扫描结果。
4. **输出到执行**：策略产出的 **`TradeIntent`** 在回测中被实例化为 **`command.TradeCommand`**（字段对齐编排层 `persistAndDispatchIntents`：`Side`、`Intent`、`TargetNotional`、`TargetPosition`、`ReduceOnly`、`DeadlineUnixMs`、`Kind=place` 等），保证命令形状与 Agent 侧一致，仅缺少真实网络投递。

## 有意近似的部分（非交易所真值）

| 模块 | 近似内容 |
|------|----------|
| 成交 | 以当期参考价（默认 K 线收）加减滑点 bps；市价为瞬间全成交/部分成交的概率模型。 |
| 手续费 | 按 maker/taker bps × 成交名义线性扣减。 |
| 资金费 | 按「持仓名义 × funding_bps_per_day × 单根 K 线时长 / 1 日」从余额扣减。 |
| 延迟 | `simulation.delay_bars` 将整条 `TradeCommand` 推迟到若干根 K 线后执行。 |
| 失败 | 独立伯努利拒绝，用于压测链路与风控统计，不拟合具体交易所错误码。 |

以上模拟的目标是 **验证数据 → Step → 命令 → 账户状态** 的闭环，而不是复刻撮合微观结构。

## 与实盘应对齐的字段小结

- **强制一致**：`AltShortStrategyInput` / `AltShortStrategyOutput`、`TradeCommand` 与 `TradeIntent` 的 JSON 形状、策略使用的特征键（如 `lr_tanh01`、`lppl_bubble_metric_01`）。
- **由回测注入、但语义对齐**：`NowUnixMs` 使用当前 K 线 `TimestampUnixMs`；`StepSequence` 单调递增；`InstanceID`/`StrategyID` 为离线占位字符串，仅占位不影响 `Step` 纯函数行为。

## 如何保证 Step 不分叉

1. **唯一实现**：业务策略计算只放在 `internal/domain/strategy`；`Step()` 作为唯一对外聚合入口（当前默认 `MinimalShortStep`）。
2. **回测包职责约束**：`internal/backtest` 只做数据加载、窗口切片、特征与 LPPL 装配、指令仿真、资金/权益与报表；**禁止**在回试包内根据「是否回测」改写阈值或重写开仓条件。
3. **代码层面的钩子**：若将来替换策略，仍然通过 **`strategy.Step`** 或 **`strategy.Stepper`** 注入，而不是新增 `BacktestStep`。

## 运行

- 配置：`configs/backtest.yaml`（数据源、`signal_lookback`、`simulation.*`、可选 `lppl`）。
- 数据：`data.provider: file` 时，`data.path` 为 CSV 文件路径，或目录（优先 `{symbol}.csv`，否则 `bars.csv`）。列为：`timestamp_ms,open,high,low,close,volume`（可选表头）。
- 命令：`go run ./cmd/backtest`（进程内使用与实盘相同的 `Step`）。
