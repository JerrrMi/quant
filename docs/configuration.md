# 配置与职责边界

AltShort 将三类进程的配置 **物理拆分** 为 `configs/saas.yaml`、`configs/agent.yaml`、`configs/backtest.yaml`，由 `internal/config` 中的 `Load*Config`、`Validate()` 装载与校验，`cmd/*` 的 `run()` 独立完成加载与上下文错误包装，再在 `internal/app` 装配 `BootstrapDeps`。

## SaaS（控制面）

**应包含：** 持久化与控制面编排相关项：数据库 DSN、Redis（控制面会话/速率/队列等占位）、对外 WebSocket 监听与路径、全局调度节拍、推理/编排侧模型标识与无量纲超参、日志级别。

**严禁包含：** Binance/API Key、API Secret、`api_key_env`/`api_secret_env` 等与交易所认证直接相关的字段；Agent 专属的本地连接细节（如出站 bind、交易所重连退避）。

**判别规则：** 若该项只为「连交易所」或「实盘下单/持仓」而存在，就不属于 SaaS。

## Agent（执行端）

**允许且应包含：** 指向密钥的 **环境变量名**（如 `ALTSHORT_BINANCE_API_KEY`，值仍在环境中）、连接 SaaS WebSocket URL、出站本地参数（bind、拨号超时、TLS 占位）、交易所名称、重连与抖动参数、风控阈值、可选本地 SQLite 状态库、日志级别。

**判别规则：** 需要触及交易所密钥或执行端网络行为的配置，默认放在 Agent；SaaS 只消费「编排所需」的非秘密元数据。

## Backtest（回放）

**应包含：** 历史数据来源（提供者、路径/清单、标的）、手续费与滑点、回放窗口与预热、`Step()` 侧模型占位参数、初始资金与币种、日志级别。

**严禁包含：** 实盘 API 密钥字段；与控制面运行时 Redis/WebSocket **服务监听** 强绑定的运行时参数（回放不需要对外暴露 SaaS 监听）。

**判别规则：** 若某项只影响 **离线数据集与仿真成本模型**，归为 Backtest；若同时影响实盘网络或密钥，拆开：秘密与执行仅在 Agent。

## 新增字段时如何判断放在哪一侧

1. **是否触及交易所密钥或签名材料？** 是 → Agent（YAML 只允许环境变量 **名**，不允许明文）。
2. **是否必须随控制面对外提供服务（DB/缓存/WS 监听/调度）？** 是 → SaaS。
3. **是否只影响历史数据加载、费用、滑点、回放窗口或模拟资金？** 是 → Backtest。
4. **是否三者都要但含义不同？** 不要复用同名字段混写；在各自 struct 下用清晰前缀或拆子节（例如 `replay.*` 仅 Backtest）。

## 代码入口

| 进程 | 配置文件 | 加载函数 | 装配函数 |
|------|-----------|----------|----------|
| SaaS | `configs/saas.yaml` | `config.LoadSaaSConfig` | `app.BootstrapSaaS` |
| Agent | `configs/agent.yaml` | `config.LoadAgentConfig` | `app.BootstrapAgent` |
| Backtest | `configs/backtest.yaml` | `config.LoadBacktestConfig` | `app.BootstrapBacktest` |

装载错误会包装为 `saas|agent|backtest config file "<path>": ...`，便于在日志中定位失败文件。
