# internal/lifecycle

进程级生命周期相关能力（优雅退出、排空、就绪探针等）的 **占位包**。

依赖装配与 `BootstrapDeps` 位于 [`internal/app/bootstrap.go`](../app/bootstrap.go)；`cmd/*` 在 `run()` 中调用 `app.BootstrapSaaS` / `BootstrapAgent` / `BootstrapBacktest`。
