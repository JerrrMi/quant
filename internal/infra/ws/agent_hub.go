package ws

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/JerrrMi/quant/internal/domain/command"
)

// AgentHub 按 AgentKey（握手 client_id）索引已连接会话，供调度器下发 command 帧。
type AgentHub struct {
	mu    sync.RWMutex
	peers map[string]*AgentSession
}

// AgentSession 绑定单个 Agent WebSocket 会话（SaaS 视角出站 seq）。
type AgentSession struct {
	AgentKey string
	Peer     *Peer
	seq      atomic.Int64
}

// NewAgentHub 创建空 Hub。
func NewAgentHub() *AgentHub {
	return &AgentHub{peers: map[string]*AgentSession{}}
}

// Register 注册或替换某 agentKey 的会话。lastOutboundSeq 为握手完成后 SaaS 在该连接上已发送的最后一帧 seq（通常为 1 的 auth 应答）。
func (h *AgentHub) Register(agentKey string, peer *Peer, lastOutboundSeq int64) (unregister func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.peers == nil {
		h.peers = map[string]*AgentSession{}
	}
	s := &AgentSession{AgentKey: agentKey, Peer: peer}
	s.seq.Store(lastOutboundSeq)
	h.peers[agentKey] = s
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if cur, ok := h.peers[agentKey]; ok && cur.Peer == peer {
			delete(h.peers, agentKey)
		}
	}
}

// SendCommand 向已连接 Agent 发送一条 command 信封（seq 为 SaaS 轴单调号）。
func (h *AgentHub) SendCommand(ctx context.Context, agentKey string, cmd command.TradeCommand) error {
	if h == nil {
		return fmt.Errorf("ws hub: nil")
	}
	h.mu.RLock()
	sess, ok := h.peers[agentKey]
	h.mu.RUnlock()
	if !ok || sess == nil || sess.Peer == nil {
		return fmt.Errorf("ws hub: no peer for agent_key %q", agentKey)
	}
	seq := sess.seq.Add(1)
	return sess.Peer.Send(ctx, MsgCommand, seq, nil, cmd)
}
