# internal/app

应用服务层：组装调度、API、回测与 domain 用例。**不放** 交易所 API 细节（下单在 `executor`）。

**Canonical**：`saas.go`、`agent.go`、`backtest.go` 中的 `Run*`。
