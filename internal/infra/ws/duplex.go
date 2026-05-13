package ws

import "context"

// RecvLoop 阻塞读取直至 ctx 取消或 handler 返回错误。
func RecvLoop(ctx context.Context, p *Peer, handler func(context.Context, *Envelope) error) error {
	for {
		env, err := p.RecvValidated(ctx)
		if err != nil {
			return err
		}
		if err := handler(ctx, env); err != nil {
			return err
		}
	}
}
