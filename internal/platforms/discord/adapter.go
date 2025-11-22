package discord

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type Adapter struct{}

func (a *Adapter) MessageConvert(data any) protocol.Message {
	return platforms.MessageConvert(data)
}

func (a *Adapter) EventConvert(data any) protocol.Event {
	var result protocol.Event

	switch evt := data.(type) {
	case *discordgo.Ready:
		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MetaEventType
		result.DetailType = protocol.MetaConnectEvent
	case *discordgo.MessageCreate:
		// Ignore all messages created by the bot itself
		if evt.Author.Bot {
			return result
		}

		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MessageEventType

		// Determine if it's a direct message or group message
		// If GuildID is empty, it's typically a DM
		topicType := "text"
		if evt.GuildID == "" {
			result.DetailType = protocol.MessageDirectEvent
			topicType = "dm"
		} else {
			result.DetailType = protocol.MessageGroupEvent
			topicType = "text"
		}

		result.Data = protocol.MessageEventData{
			Self: protocol.Self{
				Platform: ID,
			},
			MessageId:  evt.ID,
			AltMessage: evt.Content,
			UserId:     evt.Author.ID,
			TopicId:    evt.ChannelID,
			TopicType:  topicType,
		}
	case *discordgo.InteractionCreate:
		if evt.ApplicationCommandData().Name != "" {
			result.Id = types.Id()
			result.Time = time.Now().UnixMicro()
			result.Type = protocol.MessageEventType
			result.DetailType = protocol.MessageCommandEvent
			result.Data = protocol.CommandEventData{
				Command: evt.ApplicationCommandData().Name,
			}
		}
	}

	return result
}

