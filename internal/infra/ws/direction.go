package ws

import "fmt"

// Role 标明本地参与方（用于方向校验）。
type Role string

const (
	RoleAgent Role = "agent"
	RoleSaaS  Role = "saas"
)

// ValidateOutbound 校验当前 Role 是否允许发送该 MsgType。
func ValidateOutbound(role Role, typ MsgType) error {
	switch typ {
	case MsgAuth, MsgHeartbeat:
		return nil
	case MsgCommand, MsgReportAck:
		if role != RoleSaaS {
			return fmt.Errorf("ws: role %s cannot send %s", role, typ)
		}
	case MsgCommandAck, MsgDeltaReport:
		if role != RoleAgent {
			return fmt.Errorf("ws: role %s cannot send %s", role, typ)
		}
	default:
		return fmt.Errorf("ws: unknown message type %q", typ)
	}
	return nil
}

// ValidateAckCorrelation 校验应答帧携带 AckSeq（可由外层可选调用）。
func ValidateAckCorrelation(typ MsgType, ackSeq *int64) error {
	switch typ {
	case MsgCommandAck, MsgReportAck:
		if ackSeq == nil || *ackSeq <= 0 {
			return fmt.Errorf("ws: %s requires positive ack_seq", typ)
		}
	default:
		if ackSeq != nil {
			return fmt.Errorf("ws: ack_seq forbidden on type %s", typ)
		}
	}
	return nil
}
