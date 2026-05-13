# AltShort — 工程协作说明（Cursor / 人类共用）

## 一句话目标

AltShort 是一个做空山寨币的量化交易系统。

## 最高优先级架构原则（摘要）

- **`Step()` 纯函数**：策略核心计算为纯函数，便于测试与复现。
- **回测与实盘同构**：同一套调用形状贯穿回测与实盘，避免两套逻辑分叉。
- **无量纲 / 归一化计算**：指标与信号尽量在无量纲或归一化空间完成，减少跨品种硬编码。
- **SaaS 与 Agent 职责分离**：控制面、账户聚合、编排与执行端解耦。
- **Binance API Key 只存在于 Agent**：密钥不进入 SaaS 配置与进程。
- **不使用 SQL migration 文件**：数据库演进使用 GORM `AutoMigrate`，不维护 `.sql` migration 管线。

## Cursor 工作方式

- 每次修改前先阅读约束文件（本文件、`AGENTS.md`、`.cursor/rules/*`）。
- 先保证能编译，再保证能跑通最小闭环，再扩展。
- 禁止一次性重构所有模块。
- 任何涉及策略逻辑的改动都必须先保持接口不变。

## 常用命令

| 命令 | 用途 |
|------|------|
| `go test ./...` | 全量测试 |
| `go test ./... -run TestName` | 运行单个测试 |
| `go run ./cmd/saas` | 启动 SaaS 控制面 |
| `go run ./cmd/agent` | 启动执行 Agent |
| `go run ./cmd/backtest` | 启动回测 CLI |

## 禁止事项

- 不要在策略层加入网络、数据库、时间读取、文件 I/O、随机数。
- 不要把交易所密钥放入 SaaS 配置或策略包。
- 不要引入 SQL migration 文件。

## 目录导航

| 路径 | 职责 |
|------|------|
| `cmd/` | 进程入口：SaaS、Agent、回测；只做组装、配置加载与生命周期，不写策略细节。 |
| `internal/domain/` | 领域类型、值对象、策略输入输出契约；无基础设施依赖。 |
| `internal/infra/` | 日志、数据库、外部 API 客户端等可替换实现；GORM 与驱动安置于此。 |
| `internal/scheduler/` | 定时与触发编排（谁何时调用策略/执行器），不包含交易所密钥。 |
| `internal/executor/` | 下单与撤单等执行适配（实盘侧与 venue 对话）；策略逻辑不放此处。 |
| `internal/backtest/` | 历史数据装载、撮合/费用模型、回测引擎与结果收集的占位与实现。 |
| `internal/app/` | 用例编排与对外端口（HTTP 等）的粘合层（随 Phase 充实）。 |
| `internal/lifecycle/` | 进程生命周期钩子占位（优雅退出等）；依赖装配见 `internal/app/bootstrap.go`。 |

本项目另外包含 `internal/config/`（配置结构体与加载）、`docs/`（架构说明）。`migrations/` 仅占位目录，**不使用** SQL migration；请使用 GORM `AutoMigrate`。
