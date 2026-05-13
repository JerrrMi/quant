// Package ws 承载 SaaS ↔ Agent WebSocket 域协议的封装：信封、序列号、幂等与方向校验。
// 不含交易所下单逻辑；载荷类型来自 internal/domain/{auth,command,report}。
package ws
