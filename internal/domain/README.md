# internal/domain

领域类型、值对象与 **策略纯函数**（`Step`）契约。禁止依赖 `internal/infra`、网络或系统时间源（墙钟应通过 `StepInput` 注入）。

**Canonical**：`types.go`（随 Phase 扩展为更细的子文件）。
