package ws

// MsgType 为稳定的 JSON type 字面量（固定集合）。
type MsgType string

const (
	MsgAuth        MsgType = "auth"
	MsgHeartbeat   MsgType = "heartbeat"
	MsgCommand     MsgType = "command"
	MsgCommandAck  MsgType = "command_ack"
	MsgDeltaReport MsgType = "delta_report"
	MsgReportAck   MsgType = "report_ack"
)
