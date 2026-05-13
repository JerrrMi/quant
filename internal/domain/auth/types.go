// Package auth 定义 WebSocket 握手与权限边界的领域消息；不含 TLS、JWT 解析等实现。
package auth

// AuthScope 声明连接被授权的操作边界（字符串枚举，具体与 RBAC 映射由 SaaS 实现）。
type AuthScope string

const (
	// ScopeControlPlane 允许订阅编排与控制类主题（只读/读写由额外 claim 决定）。
	ScopeControlPlane AuthScope = "control_plane"
	// ScopeExecutionRead 允许读取执行回报与状态。
	ScopeExecutionRead AuthScope = "execution_read"
	// ScopeExecutionWrite 允许下发交易指令（高危）。
	ScopeExecutionWrite AuthScope = "execution_write"
)

// AuthMessage 为握手阶段客户端发往服务器的自描述载荷（密码学凭证不以此结构明文固化，仅存占位字段名）。
type AuthMessage struct {
	// ProtocolVersion 为协商标定的域协议版本号。
	ProtocolVersion string `json:"protocol_version"`

	// ClientID 为调用方标识（应用层）。
	ClientID string `json:"client_id"`

	// InstanceID 为可选：执行侧连接可绑定单一实例。
	InstanceID string `json:"instance_id,omitempty"`

	// AgentID 为可选：Agent 进程注册 id。
	AgentID string `json:"agent_id,omitempty"`

	// Token 为不透明凭证引用（实际 token 传输方式由传输层安全规范决定；可为空若仅用 mTLS）。
	Token string `json:"token,omitempty"`

	// RequestedScopes 为请求的权限范围列表。
	RequestedScopes []AuthScope `json:"requested_scopes,omitempty"`

	// Nonce 为一次性随机串，防重放（由客户端生成）。
	Nonce string `json:"nonce"`
}

// AuthResult 为服务器对 AuthMessage 的裁决（不含会话续期实现细节）。
type AuthResult struct {
	// OK 为 false 时不得进入业务帧。
	OK bool `json:"ok"`

	// SessionID 为短时会话标识（opaque）。
	SessionID string `json:"session_id,omitempty"`

	// GrantedScopes 为服务端实际授予的范围（可小于请求）。
	GrantedScopes []AuthScope `json:"granted_scopes,omitempty"`

	// Message 为拒绝原因或提示。
	Message string `json:"message,omitempty"`

	// ServerTimeUnixMs 为服务端签发时间（Unix 毫秒）。
	ServerTimeUnixMs int64 `json:"server_time_unix_ms"`
}
