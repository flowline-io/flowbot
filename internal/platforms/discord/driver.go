package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
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
	session *discordgo.Session
	stop    chan bool
}

func NewDriver(_ config.Type, _ store.Adapter) protocol.Driver {
	session, err := discordgo.New("Bot " + config.App.Platform.Discord.BotToken)
	if err != nil {
		flog.Fatal("error creating discord session: %v", err)
	}

	session.LogLevel = discordgo.LogInformational
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// register
	err = platforms.PlatformRegister(ID, &platforms.Caller{
		Action:  &Action{session: session},
		Adapter: &Adapter{},
	})
	if err != nil {
		flog.Fatal("error %v", err)
	}

	return &Driver{
		adapter: &Adapter{},
		action:  &Action{session: session},
		session: session,
		stop:    make(chan bool),
	}
}

func (d *Driver) HttpServer(_ fiber.Ctx) error {
	return nil
}

func (d *Driver) HttpWebhookClient(_ protocol.Message) error {
	return nil
}

func (d *Driver) WebSocketClient() {
	if !config.App.Platform.Discord.Enabled {
		flog.Info("Discord is disabled")
		return
	}

	err := d.session.Open()
	if err != nil {
		flog.Error(fmt.Errorf("cannot open the session: %w", err))
		return
	}

	// Ready event handler
	d.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		flog.Info("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)

		// convert and emit ready event
		protocolEvent := d.adapter.EventConvert(r)
		if protocolEvent.DetailType != "" {
			flog.Debug("start discord emit event %+v", protocolEvent)
			err := event.PublishMessage(context.Background(), protocolEvent.DetailType, protocolEvent)
			if err != nil {
				flog.Error(fmt.Errorf("failed to emit event %s, %w", protocolEvent.DetailType, err))
			}
			flog.Debug("end discord emit event %+v", protocolEvent)
		}
	})

	// Message create event handler
	d.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// convert
		protocolEvent := d.adapter.EventConvert(m)
		if protocolEvent.DetailType == "" {
			return
		}

		flog.Debug("start discord emit event %+v", protocolEvent)
		// emit event
		err := event.PublishMessage(context.Background(), protocolEvent.DetailType, protocolEvent)
		if err != nil {
			flog.Error(fmt.Errorf("failed to emit event %s, %w", protocolEvent.DetailType, err))
		}
		flog.Debug("end discord emit event %+v", protocolEvent)
	})

	// Interaction create event handler
	d.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// convert
		protocolEvent := d.adapter.EventConvert(i)
		if protocolEvent.DetailType == "" {
			return
		}

		flog.Debug("start discord emit event %+v", protocolEvent)
		// emit event
		err := event.PublishMessage(context.Background(), protocolEvent.DetailType, protocolEvent)
		if err != nil {
			flog.Error(fmt.Errorf("failed to emit event %s, %w", protocolEvent.DetailType, err))
		}
		flog.Debug("end discord emit event %+v", protocolEvent)
	})

	// Wait for stop signal
	go func() {
		<-d.stop
		flog.Info("Discord is shutting down.")
		err := d.session.Close()
		if err != nil {
			flog.Error(fmt.Errorf("failed to close discord session: %w", err))
		}
	}()
}

func (d *Driver) WebSocketServer() {
	if !config.App.Platform.Discord.Enabled {
		flog.Info("Discord is disabled")
		return
	}
}

func (d *Driver) Shoutdown() error {
	d.stop <- true
	return nil
}
