package slack

import (
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"time"
)

type Driver struct {
	adapter *Adapter
	action  *Action
	api     *slack.Client
}

func NewDriver() *Driver {
	// register
	err := platforms.PlatformRegister(ID)
	if err != nil {
		flog.Fatal(err.Error())
	}

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
		socketmode.OptionPingInterval(30*time.Second),
	)

	go func() {
		for {
			select {
			case <-stop:
				flog.Info("Slack is shutting down.")
				return
			case evt := <-client.Events:
				// ack
				switch evt.Type {
				case socketmode.EventTypeEventsAPI:
					client.Ack(*evt.Request)
				}

				// convert
				protocolEvent := d.adapter.EventConvert(evt)
				if protocolEvent.DetailType == "" {
					continue
				}

				go func() { // todo fixme
					// emit event
					err := event.Emit(protocolEvent.DetailType, types.KV{
						"caller": &platforms.Caller{
							Action:  d.action,
							Adapter: d.adapter,
						},
						"event": protocolEvent,
					})
					if err != nil {
						flog.Error(err)
					}
				}()
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
