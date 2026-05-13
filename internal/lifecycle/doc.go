// Package lifecycle 预留进程级生命周期钩子（优雅退出、排空、就绪探针等）。
//
// 依赖装配（BootstrapDeps、bootstrap.go）位于 internal/app，
// cmd/* 在完成配置加载与日志初始化后调用 app.BootstrapSaaS/Agent/Backtest。
package lifecycle
