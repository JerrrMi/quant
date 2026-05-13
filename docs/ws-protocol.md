# SaaS ↔ Agent WebSocket 域协议（v1）

本文描述控制面（SaaS）与执行端（Agent）之间的 **JSON 文本帧** 协议外壳与载荷契约。  
实现锚点：`internal/infra/ws`（信封 / seq / 方向 / Codec）、`internal/domain/{auth,command,report}`（载荷）。

**范围**：握手、心跳、指令下发与确认、执行增量上报与确认、序列号重放与幂等骨架。**不包含**开仓/平仓语义裁决（由策略与执行器负责）。

---

## 1. 帧格式

每条 WebSocket 消息为 **一条 UTF-8 JSON 对象**（建议使用 Text frame）。顶层固定包含：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 消息类型字面量（固定枚举） |
| `seq` | int64 | 是 | **发送方**单调递增序号，从 `1` 开始；每条出站消息 +1 |
| `ack_seq` | int64 | 条件 | 仅应答类消息必填：指向 **被确认帧在其发送方上的 `seq`** |
| `payload` | object | 是 | 业务载荷（随 `type` 变化） |

约定两端各自维护一条出站序号轴：**SaaS 轴**与 **Agent 轴**互不混用；引用对端序号时使用 `ack_seq`。

---

## 2. 消息类型列表与方向

| `type` | 允许发送方 | 允许接收方 |
|--------|------------|------------|
| `auth` | SaaS、Agent | SaaS、Agent |
| `heartbeat` | SaaS、Agent | SaaS、Agent |
| `command` | SaaS | Agent |
| `command_ack` | Agent | SaaS |
| `delta_report` | Agent | SaaS |
| `report_ack` | SaaS | Agent |

首次业务帧之前必须先完成 **双向 `auth`**（见 §4）。

---

## 3. 载荷必填字段

### 3.1 `auth`（Agent → SaaS）— `AuthMessage`

| 字段 | 必填 |
|------|------|
| `protocol_version` | 是 |
| `client_id` | 是 |
| `nonce` | 是 |
| `instance_id` / `agent_id` / `token` / `requested_scopes` | 否 |
| `last_seen_saas_seq` | 否（重连收敛，见 §7） |

### 3.2 `auth`（SaaS → Agent）— `AuthResult`

| 字段 | 必填 |
|------|------|
| `ok` | 是 |
| `server_time_unix_ms` | 是 |
| `session_id` / `granted_scopes` | `ok=true` 时常填 |
| `error_code` / `message` | `ok=false` 时应填 |
| `last_seen_agent_seq` | 否（镜子字段，便于对账） |

### 3.3 `heartbeat` — `HeartbeatPayload`

| 字段 | 必填 |
|------|------|
| `ts_unix_ms` | 是 |

### 3.4 `command` — `TradeCommand`

| 字段 | 必填 |
|------|------|
| `command_id` | 是 |
| `instance_id` | 是 |
| `strategy_id` | 是 |
| `symbol` | 是 |
| `side` | 是 |
| `intent` | 是 |
| `reduce_only` | 是 |
| `deadline` | 是（Unix 毫秒） |
| `nonce` | 是 |
| `idempotency_key` | 是 |
| `kind` | 是 |
| `target_notional` / `target_position` | 条件（至少其一或与 `intent` 组合可满足执行前置条件；执行文档细化） |

协议层 **不解释** `side` / `intent` 的业务语义（不问开仓或平仓）。

### 3.5 `command_ack` — `CommandAck`

| 字段 | 必填 |
|------|------|
| `command_id` | 是 |
| `status` | 是 |
| `agent_time_unix_ms` | 是 |
| `ref_envelope_seq` | 强烈建议（与被确认的 `command.seq` 对齐） |
| 外层 `ack_seq` | **必填**，等于对应的 SaaS `command.seq` |

### 3.6 `delta_report` — `DeltaReport`

| 字段 | 必填 |
|------|------|
| `report_id` | 是 |
| `instance_id` | 是 |

可选块（按需填充，数组可空）：`fills`、`open_orders`、`positions`、`account`、`errors`、`details`、`exchange_event_time_unix_ms`。  
分别承载：**成交回报**、**未成交订单快照**、**持仓快照**、**权益 / 保证金摘要**、**归一化错误**。

### 3.7 `report_ack` — `ReportAck`

| 字段 | 必填 |
|------|------|
| `report_id` | 是 |
| `received` | 是 |
| `server_time_unix_ms` | 是 |
| `ref_envelope_seq` | 强烈建议 |
| 外层 `ack_seq` | **必填**，等于对应的 Agent `delta_report.seq` |

---

## 4. 握手流程

1. Agent 发送 `auth`（`seq=1`），载荷 `AuthMessage`。  
2. SaaS 校验凭证与 RBAC；若拒绝：`auth` 回应 `AuthResult.ok=false`，关闭连接或停留只读策略（实现自定）。  
3. 若接受：`auth` 回应 `AuthResult.ok=true`，授予 `granted_scopes`。  
4. 仅在 `ok=true` 之后允许 `command` / `delta_report` 等业务类型。

---

## 5. 幂等规则

### 5.1 指令（`command`）

- **自然键**：优先 `idempotency_key`；若为空则退化 `command_id`（二者至少一条在 SaaS 真源侧保证稳定）。
- Agent 进程内维护「已适用指令」集合：**同一自然键第二次到达不得再次进入执行管线**（网络重传 / SaaS 重放同源）。
- SaaS 侧对未完成指令亦可基于同一自然键去重发布。

### 5.2 回报（`delta_report`）

- `report_id` 在 Agent 生成侧应尽量唯一；SaaS 对同一 `report_id` 重复投递可做幂等丢弃或覆盖策略（存储层约定）。
- **成交**：按 `FillRecord.fill_id` 去重合并；重复 `fill_id` **不得**加倍仓位或流水。
- 辅助函数：`internal/infra/ws.MergeFillSnapshots`。

### 5.3 `command_ack` / `report_ack`

- 必须能通过 `command_id` / `report_id` **以及** `ack_seq` ↔ 原始 `seq` 双重关联到原始帧。

---

## 6. 重放规则（SaaS → Agent）

当 SaaS 怀疑 Agent 丢帧（未收到 `command_ack`）或 Agent 重连：

1. Agent 在握手 `AuthMessage.last_seen_saas_seq` 填入：**已成功处理的来自 SaaS 的最大 `seq`**（包含已 ack 的 command / report_ack / auth / heartbeat 是否计入须两端一致，建议 **所有 SaaS 出站帧均计入**）。
2. SaaS 侧缓冲（或日志）中选取所有 `seq > last_seen_saas_seq` 的出站信封 **按序重放**。
3. Agent 对重放帧执行与普通帧相同的 **幂等过滤**：已适用的 `idempotency_key` 不再触发执行。

判定辅助：`internal/infra/ws.SaasOutboundNeedsReplay(last_seen, envelope_seq)`。

**反向（Agent → SaaS）**：`delta_report` 重传同样依赖 `report_id` / `fill_id` 幂等与 SaaS 侧去重。

---

## 7. 断线重连后的状态收敛（骨架）

推荐顺序：

1. **TCP/TLS + WS** 重建。
2. Agent `auth`，附带 `last_seen_saas_seq`；可选附带实例维度标识。
3. SaaS `AuthResult` 回应并可镜像 `last_seen_agent_seq`。
4. SaaS **重放丢失指令窗口**（§6）。
5. Agent **全量或对账式推送**最新 `delta_report`（持仓 / 挂单 / 权益 / 近期 fills）；SaaS `report_ack`。
6. 双方心跳恢复；超时策略由运维配置。

---

## 8. `last_seen_saas_seq` 协作要点

| 角色 | 字段 | 语义 |
|------|------|------|
| Agent → SaaS | `last_seen_saas_seq` | 「我已消费的 SaaS 出站最大 `seq`」 |
| SaaS → Agent（可选） | `last_seen_agent_seq` | 「我已记录的 Agent 出站最大 `seq`」镜像 |

重放边界：**严格大于**该水位线的 SaaS 帧才需要再次投递给 Agent。

---

## 9. JSON 稳定性

- 字段名由 Go struct `json` tag 锁定；两端共用 `internal/domain` 类型。
- 新增字段须 **向后兼容**（旧实现忽略未知字段）。
- **禁止**在协议层根据载荷推断开仓/平仓；仅传递已由编排产生的 `side` / `intent`。

---

## 10. 参考类型映射

| `type` | Go payload |
|--------|------------|
| `auth` | `internal/domain/auth.AuthMessage` / `AuthResult` |
| `heartbeat` | `internal/infra/ws.HeartbeatPayload` |
| `command` | `internal/domain/command.TradeCommand` |
| `command_ack` | `internal/domain/command.CommandAck` |
| `delta_report` | `internal/domain/report.DeltaReport` |
| `report_ack` | `internal/domain/report.ReportAck` |

验收用例：`go test ./internal/infra/ws -run TestMinimal`。
