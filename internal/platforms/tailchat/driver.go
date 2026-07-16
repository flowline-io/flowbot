package tailchat

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// webhookTokenHeader is the shared-secret header for inbound Tailchat callbacks.
const webhookTokenHeader = "X-Tailchat-Token"

type Driver struct {
	adapter *Adapter
	action  *Action
	stop    chan bool
}

func NewDriver(_ *config.Type, _ store.Adapter) protocol.Driver {
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

	if err := verifyTailchatWebhookToken(ctx); err != nil {
		return err
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

// verifyTailchatWebhookToken requires platform.tailchat.webhook_token and a matching header.
func verifyTailchatWebhookToken(ctx fiber.Ctx) error {
	expected := config.App.Platform.Tailchat.WebhookToken
	if expected == "" {
		return types.Errorf(types.ErrUnauthorized, "tailchat webhook token not configured")
	}
	provided := ctx.Get(webhookTokenHeader)
	if provided == "" {
		return types.Errorf(types.ErrUnauthorized, "missing %s header", webhookTokenHeader)
	}
	if !secureTokenEqual(provided, expected) {
		return types.Errorf(types.ErrUnauthorized, "invalid webhook token")
	}
	return nil
}

// secureTokenEqual compares a and b without leaking length via early return.
func secureTokenEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

func (*Driver) HttpWebhookClient(_ protocol.Message) error {
	return nil
}

func (*Driver) WebSocketClient() {
	if !config.App.Platform.Tailchat.Enabled {
		flog.Info("Tailchat is disabled")
		return
	}
}

func (*Driver) WebSocketServer() {
	if !config.App.Platform.Tailchat.Enabled {
		flog.Info("Tailchat is disabled")
		return
	}
}

func (d *Driver) Shutdown() error {
	d.stop <- true
	return nil
}
