# AltShort 验收矩阵（工程底座）

本文档是 **「可长期迭代」** 的逐项验收表：以系统约束与关键路径为主，不以覆盖率为目标。  
自动化验证入口：`go test ./...`（**禁止**依赖真实交易所、**禁止**不可控墙钟断言）。

| ID | 验收项 | 约束 / 风险 | 自动验证（测试或命令） | 状态 |
|----|--------|-------------|------------------------|------|
| A1 | **SaaS 可启动** | 控制面进程可组装依赖并进入运行态 | `go build -o NUL ./cmd/saas`（或本地 `go run ./cmd/saas` 冒烟） | 手动 / CI 编译 |
| A2 | **Agent 可启动** | 执行面进程可组装；密钥仅在 Agent 配置注入 | `go build -o NUL ./cmd/agent` | 手动 / CI 编译 |
| A3 | **回测可启动** | 回测 CLI 可加载配置并跑通文件数据源 | `go test ./internal/backtest/...`；`go build ./cmd/backtest` | 已测 |
| A4 | **Step 可独立测试** | `Step()` 纯函数：无 I/O、无墙钟、确定性 I/O 形状 | `go test ./internal/domain/strategy/...`（含 JSON 快照、空输入、边界、意图分支） | 已测 |
| A5 | **命令可下发、确认、回报** | SaaS→Agent JSON 信封；`command_ack` 带 `ack_seq`；`delta_report` / `report_ack` | `go test ./internal/infra/ws/...` | 已测 |
| A6 | **幂等与重放安全** | 同一 `IdempotencyKey` 不应二次下单；重放帧被进程内/持久 dedup 消化 | `TestReconnectReplayDoesNotReexecuteCommand`；`TestService_idempotentReplaySkipsSecondPlace` | 已测 |
| A7 | **断线恢复语义** | 会话内去重表可重置 vs 持久幂等镜像；缓冲 seq 重放条件明确 | `TestCommandDedup_dropSeenModelsSessionReset`；`TestSaasOutboundNeedsReplay_semantics` | 已测 |
| A8 | **执行器失败与超时** | 截止期拒绝；可重试 API 错误 bounded backoff；`context` 取消快速失败 | `TestService_CommandExpiredDeadline`；`TestService_placeRetriesOnRetryableAPIError`；`TestService_contextCancelBeforeVenueWork` | 已测 |
| A9 | **回报上报** | `DeltaReport` 含账户/仓位聚合（refreshSnapshot） | `TestService_deltaReportIncludesAccountSnapshot` | 已测 |
| A10 | **数据库演进** | 仅 GORM `AutoMigrate`；可重复迁移 | `go test ./internal/infra/db/...`；`TestGormInstanceRepository_CreateGetRoundTrip`；`TestSQLiteFile_survivesReconnect` | 已测 |
| A11 | **回测 ≈ 实盘输入形状** | 同一 `strategy.Step`；意图→`TradeCommand` 字段一致 | `TestEngine_tradeCommandFromIntent_liveIsomorphism`；`TestBuildInput_usesSharedStep` | 已测 |
| A12 | **成本与执行模拟** | 手续费模型可观测、可导出报告 | `TestSimulator_takerFeeOnFilledSell`；`TestBacktestReport_JSONExportStable`；`TestEngine_runAccumulatesFeesWhenTradesFire` | 已测 |
| A13 | **状态可自动收敛** | 调度编排、重连策略（不含交易所） | `go test ./internal/scheduler/...`；`go test ./internal/lifecycle/...` | 已测 |
| A14 | **策略与执行职责不混淆** | 策略包不依赖 `executor`/`infra`；执行层不含择时逻辑 | `go build ./...` + 包 import 边界评审（见 `CLAUDE.md` / `docs/strategy-contract.md`） | 持续评审 |

## Phase 与测试映射（快速反馈）

| Phase 典型改动 | 优先跑 |
|----------------|--------|
| 策略契约、`Step`、特征归一化 | `go test ./internal/domain/strategy/...` |
| WebSocket 协议、命令/回报模型 | `go test ./internal/infra/ws/...` |
| Agent 执行、幂等、重试 | `go test ./internal/executor/...` |
| 回测引擎、费用、报告 | `go test ./internal/backtest/...` |
| 模型与仓储 | `go test ./internal/infra/db/...` |

## 明确非目标

- 本矩阵 **不** 要求联机 Binance 作为 CI 前置。
- 本矩阵 **不** 将「实现细节逐项镜像测试」作为成功标准；只锁定对外契约与架构约束。
