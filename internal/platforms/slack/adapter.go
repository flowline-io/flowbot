package slack

import (
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"time"
)

type Adapter struct{}

func (a *Adapter) MessageConvert(data any) protocol.Message {
	// todo slack message -> protocol message
	return platforms.MessageConvert(data)
}

func (a *Adapter) EventConvert(data any) protocol.Event {
	var result protocol.Event
	event, ok := data.(socketmode.Event)
	if !ok {
		return result
	}

	switch event.Type {
	// connect
	case socketmode.EventTypeHello:
		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MetaEventType
		result.DetailType = protocol.MetaConnectEvent
	// event
	case socketmode.EventTypeEventsAPI:
		apiEvent := event.Data.(slackevents.EventsAPIEvent)

		switch apiEvent.InnerEvent.Type {
		case "message":
			messageEvent := apiEvent.InnerEvent.Data.(*slackevents.MessageEvent)

			// Ignore all messages created by the bot itself
			if messageEvent.BotID != "" {
				return result
			}

			result.Id = types.Id()
			result.Time = time.Now().UnixMicro()
			result.Type = protocol.MessageEventType
			if messageEvent.ChannelType == "im" {
				result.DetailType = protocol.MessageDirectEvent
			}
			if messageEvent.ChannelType == "channel" {
				result.DetailType = protocol.MessageGroupEvent
			}

			// todo message data
			result.Data = protocol.MessageEventData{
				Self: protocol.Self{
					Platform: ID,
					UserId:   "", // todo
				},
				MessageId:  messageEvent.ClientMsgID,
				Message:    a.MessageConvert(messageEvent),
				AltMessage: messageEvent.Text,
				UserId:     messageEvent.User,
				TopicId:    messageEvent.Channel,
				TopicType:  messageEvent.ChannelType, // todo
			}
		}

	// slash command
	case socketmode.EventTypeSlashCommand:
		cmd, ok := event.Data.(slack.SlashCommand)
		if !ok {
			return result
		}

		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MessageEventType
		result.DetailType = protocol.MessageCommandEvent
		result.Data = protocol.CommandEventData{
			Command: cmd.Command,
		}
	}

	return result
}
