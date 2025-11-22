package tailchat

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
)

type Driver struct {
	adapter *Adapter
	action  *Action
	stop    chan bool
}

func NewDriver(_ config.Type, _ store.Adapter) protocol.Driver {
	client := newClient()

	// register
	err := platforms.PlatformRegister(ID, &platforms.Caller{
		Action:  &Action{client: client},
		Adapter: &Adapter{},
	})
	if err != nil {
		flog.Fatal("error %v", err)
	}

	return &Driver{
		adapter: &Adapter{},
		action:  &Action{client: client},
		stop:    make(chan bool),
	}
}

func (d *Driver) HttpServer(ctx fiber.Ctx) error {
	if !config.App.Platform.Tailchat.Enabled {
		flog.Info("Tailchat is disabled")
		return nil
	}

	var payload Payload
	err := ctx.Bind().Body(&payload)
	if err != nil {
		return err
	}

	// convert
	protocolEvent := d.adapter.EventConvert(&payload)
	if protocolEvent.DetailType == "" {
		return nil
	}

	flog.Debug("start tailchat emit event %+v", protocolEvent)
	// emit event
	err = event.PublishMessage(context.Background(), protocolEvent.DetailType, protocolEvent)
	if err != nil {
		flog.Error(fmt.Errorf("failed to emit event %s, %w", protocolEvent.DetailType, err))
		return err
	}
	flog.Debug("end tailchat emit event %+v", protocolEvent)

	return nil
}

func (d *Driver) HttpWebhookClient(_ protocol.Message) error {
	return nil
}

func (d *Driver) WebSocketClient() {
	if !config.App.Platform.Tailchat.Enabled {
		flog.Info("Tailchat is disabled")
		return
	}
}

func (d *Driver) WebSocketServer() {
	if !config.App.Platform.Tailchat.Enabled {
		flog.Info("Tailchat is disabled")
		return
	}
}

func (d *Driver) Shoutdown() error {
	d.stop <- true
	return nil
}
