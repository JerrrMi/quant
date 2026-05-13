package ws

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
)

// WebsocketTransport 将 gorilla WebSocket 适配为文本 JSON 帧传输。
type WebsocketTransport struct {
	Conn *websocket.Conn
}

// ReadFrame 读取一条文本帧。
func (w *WebsocketTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	mt, data, err := w.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("ws: unexpected websocket frame type %d", mt)
	}
	return data, nil
}

// WriteFrame 写入一条 UTF-8 JSON 文本帧。
func (w *WebsocketTransport) WriteFrame(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return w.Conn.WriteMessage(websocket.TextMessage, data)
}

// Close 关闭底层连接。
func (w *WebsocketTransport) Close() error {
	return w.Conn.Close()
}
