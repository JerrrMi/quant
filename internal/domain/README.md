# internal/domain

领域类型与值对象；**策略纯函数**契约见 `strategy` 子包（`Stepper`）。禁止依赖 `internal/infra`、网络或系统时间源（墙钟通过 `AltShortStrategyInput.NowUnixMs` 注入）。

**Canonical**：根 `types.go`（`Bar`、`Side`）；冻结模型见 `docs/domain-model.md`。
