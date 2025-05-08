package slack

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"time"
)

type Driver struct {
	adapter *Adapter
	action  *Action
	api     *slack.Client
	stop    chan bool
}

func NewDriver(_ config.Type, _ store.Adapter) protocol.Driver {
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
		flog.Fatal("error %v", err)
	}

	return &Driver{
		adapter: &Adapter{},
		action:  &Action{api: api},
		api:     api,
		stop:    make(chan bool),
	}
}

func (d *Driver) HttpServer(_ *fiber.Ctx) error {
	return nil
}

func (d *Driver) HttpWebhookClient(_ protocol.Message) error {
	return nil
}

func (d *Driver) WebSocketClient() {
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
			case <-d.stop:
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
				err := event.PublishMessage(context.Background(), protocolEvent.DetailType, protocolEvent)
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

func (d *Driver) WebSocketServer() {
	if !config.App.Platform.Slack.Enabled {
		flog.Info("Slack is disabled")
		return
	}
}

func (d *Driver) Shoutdown() error {
	d.stop <- true
	return nil
}
