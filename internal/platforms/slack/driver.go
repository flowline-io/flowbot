package slack

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
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

	// register
	err := platforms.PlatformRegister(ID, &platforms.Caller{
		Action:  &Action{api: api},
		Adapter: &Adapter{},
	})
	if err != nil {
		flog.Fatal(err.Error())
	}

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

				flog.Debug("start slack emit event %+v", protocolEvent)
				// emit event
				err := event.PublishMessage(protocolEvent.DetailType, protocolEvent)
				if err != nil {
					flog.Error(fmt.Errorf("failed to emit event %s, %w", protocolEvent.DetailType, err))
				}
				flog.Debug("end slack emit event %+v", protocolEvent)
			}
		}
	}()

	go func() {
		err := client.Run()
		if err != nil {
			flog.Error(fmt.Errorf("failed to run socket mode, %w", err))
		}
	}()
}

func (d *Driver) WebSocketServer(_ <-chan bool) {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}
}
