package slack

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type Driver struct {
	adapter Adapter
}

func NewDriver() *Driver {
	return &Driver{
		adapter: Adapter{},
	}
}

func (d *Driver) HttpServer(ctx *fiber.Ctx) error {
	return nil
}

func (d *Driver) HttpWebhookClient(message protocol.Message) error {
	return nil
}

func (d *Driver) WebSocketClient(stop <-chan bool) {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}

	api := slack.New(
		config.App.Platform.Slack.BotToken,
		slack.OptionDebug(true),
		slack.OptionAppLevelToken(config.App.Platform.Slack.AppToken),
	)
	client := socketmode.New(api, socketmode.OptionDebug(true))

	go func() {
		for {
			select {
			case <-stop:
				flog.Info("Slack is shutting down.")
				return
			case event := <-client.Events:
				// convert
				protocolEvent := d.adapter.EventConvert(event)
				if protocolEvent.DetailType == "" {
					continue
				}

				// emit event
				pkgEvent.AsyncEmit(protocolEvent.DetailType, types.KV{"event": protocolEvent})

				// ack
				if protocolEvent.Type == protocol.MessageEventType {
					client.Ack(*event.Request)
				}
			}
		}
	}()

	go func() {
		err := client.Run()
		if err != nil {
			flog.Error(err)
		}
	}()
}

func (d *Driver) WebSocketServer(stop <-chan bool) {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}
}

// SlackRequest takes in the StatusCode and Content from other functions to display to the user's slack.
type SlackRequest struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content string `json:"body"`
	// Channel is the channel that the message will be sent to.
	Channel string `json:"channel"`
}
