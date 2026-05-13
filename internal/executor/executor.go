// Package executor 将领域意图转换为交易所 API 调用；只在 Agent 进程构造与运行。
package executor

import "context"

// ExchangeExecutor 封装与具体交易所 REST/WebSocket 的交互。
// 密钥材料由上层通过环境或安全存储注入，不在结构体字段中明文保存。
type ExchangeExecutor struct {
	// Venue 是场所名，如 "binance"；用于路由具体 client 实现。
	Venue string
}

// NewExchangeExecutor 创建执行器占位；未来注入签名、HTTP 客户端等。
// 仅由 cmd/agent 或 app/agent 路径调用。
func NewExchangeExecutor(venue string) *ExchangeExecutor {
	return &ExchangeExecutor{Venue: venue}
}

// Ping 占位连通性检查；后续替换为真实余额/时间端点探测。
func (e *ExchangeExecutor) Ping(ctx context.Context) error {
	_ = e
	_ = ctx
	return nil
}
