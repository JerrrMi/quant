package ws

import (
	"context"
	"time"
)

// HeartbeatPayload 应用层心跳载荷（可与 WS ping/pong 并存）。
type HeartbeatPayload struct {
	TsUnixMs int64 `json:"ts_unix_ms"`
}

// DefaultHeartbeatInterval 占位默认值；真实调度由调用方注入。
const DefaultHeartbeatInterval = 30 * time.Second

// RunHeartbeatTicker 仅在 ctx 存活期间周期性触发 send（骨架：不包含自动读应答）。
func RunHeartbeatTicker(ctx context.Context, interval time.Duration, send func(tsUnixMs int64) error) error {
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := send(time.Now().UnixMilli()); err != nil {
				return err
			}
		}
	}
}
