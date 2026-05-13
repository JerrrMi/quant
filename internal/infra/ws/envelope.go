package ws

import "encoding/json"

// Envelope 为线上帧的稳定外壳：每条消息必有发送方单调 seq；应答帧通过 AckSeq 指向对端被确认帧。
type Envelope struct {
	Type MsgType `json:"type"`
	Seq  int64   `json:"seq"`

	// AckSeq 在被确认帧的发送方 seq（例如 Agent 确认 SaaS command.seq）。
	AckSeq *int64 `json:"ack_seq,omitempty"`

	Payload json.RawMessage `json:"payload"`
}
