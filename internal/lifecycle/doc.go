// Package lifecycle organizes process lifecycle: ordered startup/shutdown, reconnect backoff helpers,
// and graceful shutdown steps. It deliberately excludes strategy and execution-domain decisions.
//
// Operational semantics (startup order, reconnect, shutdown, persisted state) are documented in docs/lifecycle.md.
//
// BootstrapDeps remain owned by internal/app; cmd loads configs then calls BootstrapSaaS/Agent/Backtest before runtime loops.
package lifecycle
