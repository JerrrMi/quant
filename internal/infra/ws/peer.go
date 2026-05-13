package ws

import (
	"context"
	"fmt"
)

// FrameTransport 抽象一条完整 UTF-8 JSON 消息的收发边界。
type FrameTransport interface {
	ReadFrame(ctx context.Context) ([]byte, error)
	WriteFrame(ctx context.Context, data []byte) error
	Close() error
}

// Peer 绑定本地 Role、传输与 Codec；负责出站校验。
type Peer struct {
	Role      Role
	Transport FrameTransport
	Codec     Codec
}

// NewPeer 构造 Peer；Codec 为空时使用 JSONCodec。
func NewPeer(role Role, t FrameTransport, c Codec) *Peer {
	if c == nil {
		c = JSONCodec{}
	}
	return &Peer{Role: role, Transport: t, Codec: c}
}

// Send 序列化并写入；校验出站方向与 ack_seq 规则。
func (p *Peer) Send(ctx context.Context, typ MsgType, seq int64, ackSeq *int64, payload any) error {
	if err := ValidateOutbound(p.Role, typ); err != nil {
		return err
	}
	if err := ValidateAckCorrelation(typ, ackSeq); err != nil {
		return err
	}
	raw, err := p.Codec.MarshalEnvelope(typ, seq, ackSeq, payload)
	if err != nil {
		return err
	}
	return p.Transport.WriteFrame(ctx, raw)
}

// Recv 读取并解码信封外壳（不含方向校验）。
func (p *Peer) Recv(ctx context.Context) (*Envelope, error) {
	raw, err := p.Transport.ReadFrame(ctx)
	if err != nil {
		return nil, err
	}
	return p.Codec.UnmarshalEnvelope(raw)
}

// RecvValidated 在读帧后校验：①入站方向；② ack_seq 携带规则。
func (p *Peer) RecvValidated(ctx context.Context) (*Envelope, error) {
	env, err := p.Recv(ctx)
	if err != nil {
		return nil, err
	}
	if err := ValidateInbound(p.Role, env.Type); err != nil {
		return nil, err
	}
	if err := ValidateAckCorrelation(env.Type, env.AckSeq); err != nil {
		return nil, err
	}
	return env, nil
}

// ValidateInbound 校验 receiver 视角允许的对端发送类型。
func ValidateInbound(receiver Role, typ MsgType) error {
	switch receiver {
	case RoleAgent:
		switch typ {
		case MsgAuth, MsgHeartbeat, MsgCommand, MsgReportAck:
			return nil
		}
	case RoleSaaS:
		switch typ {
		case MsgAuth, MsgHeartbeat, MsgCommandAck, MsgDeltaReport:
			return nil
		}
	default:
		return fmt.Errorf("ws: unknown receiver role %q", receiver)
	}
	return fmt.Errorf("ws: receiver role %s rejects inbound type %s", receiver, typ)
}
