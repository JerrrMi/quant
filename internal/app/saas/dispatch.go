package saas

import (
	"context"
	"fmt"

	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/ws"
	"github.com/JerrrMi/quant/internal/scheduler"
)

// HubDispatcher 实现 scheduler.CommandDispatcher，将指令送到已连接 Agent WebSocket。
type HubDispatcher struct {
	Hub *ws.AgentHub
}

func (d *HubDispatcher) Dispatch(ctx context.Context, instance *models.Instance, cmd command.TradeCommand) error {
	if d == nil || d.Hub == nil {
		return fmt.Errorf("hub dispatcher: no hub")
	}
	if instance == nil {
		return fmt.Errorf("hub dispatcher: nil instance")
	}
	return d.Hub.SendCommand(ctx, instance.AgentKey, cmd)
}

var _ scheduler.CommandDispatcher = (*HubDispatcher)(nil)
