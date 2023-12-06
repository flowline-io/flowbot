package slack

import (
	"github.com/flowline-io/flowbot/internal/platforms"
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
	adapter *Adapter
	action  *Action
	api     *slack.Client
}

func NewDriver() *Driver {
	api := slack.New(
		config.App.Platform.Slack.BotToken,
		slack.OptionDebug(config.App.Log.Level == flog.DebugLevel),
		slack.OptionLog(flog.SlackLogger),
		slack.OptionAppLevelToken(config.App.Platform.Slack.AppToken),
	)
	return &Driver{
		adapter: &Adapter{},
		action:  &Action{api: api},
		api:     api,
	}
}

func (d *Driver) HttpServer(_ *fiber.Ctx) error {
	return nil
}

func (d *Driver) HttpWebhookClient(_ protocol.Message) error {
	return nil
}

func (d *Driver) WebSocketClient(stop <-chan bool) {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}

	client := socketmode.New(
		d.api,
		socketmode.OptionDebug(config.App.Log.Level == flog.DebugLevel),
		socketmode.OptionLog(flog.SlackLogger),
	)

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
				err := pkgEvent.Emit(protocolEvent.DetailType, types.KV{
					"caller": &platforms.Caller{
						Action:  d.action,
						Adapter: d.adapter,
					},
					"event": protocolEvent,
				})
				if err != nil {
					flog.Error(err)
					continue
				}

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

func (d *Driver) WebSocketServer(_ <-chan bool) {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}
}
