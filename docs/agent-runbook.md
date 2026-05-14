# AltShort Agent Runbook（执行端运维）

进程入口：`cmd/agent`。实现：`internal/app/agent`、`internal/executor`、`internal/infra/binance`。Agent **不做策略计算**，只做连接、下单/撤单/查询与回报聚合。

---

## 1. 前置条件

- **配置文件**：默认 `configs/agent.yaml`（等价于你说的 `config.agent.yaml`——请任选其一作为真源路径并在部署编排里固定）。
- **环境变量**：`binance.api_key_env`、`binance.api_secret_env` **只是变量名**。真实密钥只存在于运行时环境／密钥管理服务，不写回仓库。
- **SaaS 可达**：仅能连 `connection.saas_ws_url`；Agent **绝不**读取 SaaS 私有数据库 DSN。
- **身份对齐**：`identity.client_id` 与控制面分配给实例的 **`AgentKey` 完全一致**，否则 SaaS Hub 路由不到会话，命令无法投递。
- **交易所**：当前仅实现 **Binance USD-M Futures REST**（`exchange.name=binance`）；`binance.use_testnet` 切换 REST 基底 URL。
- **本地幂等库**：`local_store.sqlite_dsn` 为空则进程内存 SQLite（重启丢幂等表）。生产建议使用独立 SQLite 文件，保证 `IdempotencyKey` 在重启后继续有效。

---

## 2. 必须保密的配置项

- API **Key / Secret**、以及占位字段 `passphrase_env`（非 Binance 场景）只能出现在 Agent 宿主环境或其密钥注入管道。
- **禁止**出现在 `configs/saas.yaml`、`cmd/saas` 依赖链、策略包或任意 Git 跟踪文件。
- 日志请勿输出签名原文、`X-MBX-APIKEY` 值或 URL 中签名的 query。

---

## 3. 如何识别连接异常

- **WebSocket**：读帧失败、`EOF`、或对端断开 → 当前会话退出；外层循环退避并重拨。
- **Auth**：首帧后若 `AuthResult.ok=false`，进程报错退出并由进程管理器重拉。
- **交易所**：单笔命令失败会在 `delta_report.details["retry_hint"]`（若可归类）中出现 `rate_limit`、`resync_timestamp` 等；HTTP 418/429/5xx 与 Binance 业务码 `-1003`、`-1021` 等会触发带退避的重试（见 `internal/executor/service.go`）。

---

## 4. 重连

- 协议级 `last_seen_saas_seq`：`AuthMessage.last_seen_saas_seq` **跨重连递增持久化**，记录 Agent 已成功消费的 SaaS 出站最大 `seq`；SaaS 侧按需重放的出站缓冲仍为后续 Phase（见 `docs/ws-protocol.md`）。
- **`reconnect.max_attempts`**：允许的 **累计 WebSocket 会话次数**（每建立一次会话即计 1 次，包含进程启动时的首次连接）；`0` 为不限制，直到进程退出。
- `initial_backoff_seconds` / `max_backoff_seconds` / `jitter_ratio`：会话结束后睡眠时间，指数增加并封顶。

---

## 5. 如何确认命令是否执行成功

- 查 **`command_ack`**：`status`、`exchange_order_id`、`ref_envelope_seq`（应等于 SaaS `command.seq`）。
- 查紧随其后 **`delta_report`**：`fills`、`open_orders`、`positions`、`account`、`errors`。`report_id` 每次换新 UUID。
- 交易所在 Binance UI 可按 **Client Order Id**（由 `IdempotencyKey`/`command_id` 清洗得到）对齐 REST 返回值。

---

## 6. 如何恢复状态

- **指令幂等**：`AgentDedupKey`（GORM，`internal/infra/db/models`）记在 **Agent 本地库**；重放同名 `IdempotencyKey` 时优先 `GET /fapi/v1/order` 拉平而非再次 `POST /order`。
- **交易所侧重复**：若 `POST /order` 报重复语义，适配层会降级为订单查询后继续汇报。
- **策略与模型**：Agent 不缓存策略权重或模型推理参数；一切以 SaaS 下发 `TradeCommand` 为准。

---

## 7. 优雅退出

`cmd/agent` 使用 `signal.NotifyContext` 监听 **Ctrl+C** 与 SIGTERM（类 Unix）；取消后读完当前帧即退出重连循环。

---

参见：`docs/ws-protocol.md`、`docs/data-ownership.md`。
