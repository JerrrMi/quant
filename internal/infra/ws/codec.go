package ws

import (
	"encoding/json"
	"fmt"
)

// Codec 定义稳定的 JSON 帧编解码（UTF-8 JSON text frame）。
type Codec interface {
	MarshalEnvelope(typ MsgType, seq int64, ackSeq *int64, payload any) ([]byte, error)
	UnmarshalEnvelope(data []byte) (*Envelope, error)
}

// JSONCodec 为标准 encoding/json 实现；字段名由 domain 类型 json tag 锁定。
type JSONCodec struct{}

// MarshalEnvelope 序列化整条信封。
func (JSONCodec) MarshalEnvelope(typ MsgType, seq int64, ackSeq *int64, payload any) ([]byte, error) {
	if seq <= 0 {
		return nil, fmt.Errorf("ws: envelope seq must be positive")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ws: marshal payload: %w", err)
	}
	env := Envelope{
		Type:    typ,
		Seq:     seq,
		AckSeq:  ackSeq,
		Payload: raw,
	}
	out, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("ws: marshal envelope: %w", err)
	}
	return out, nil
}

// UnmarshalEnvelope 解码信封外壳。
func (JSONCodec) UnmarshalEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("ws: unmarshal envelope: %w", err)
	}
	if env.Type == "" || env.Seq <= 0 {
		return nil, fmt.Errorf("ws: invalid envelope type or seq")
	}
	return &env, nil
}

// DecodePayload 将 Payload 解码到具体领域类型。
func DecodePayload[T any](env *Envelope, dst *T) error {
	if env == nil {
		return fmt.Errorf("ws: nil envelope")
	}
	if err := json.Unmarshal(env.Payload, dst); err != nil {
		return fmt.Errorf("ws: decode payload: %w", err)
	}
	return nil
}
