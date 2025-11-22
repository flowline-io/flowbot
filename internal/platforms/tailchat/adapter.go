package tailchat

import (
	"time"

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
	case *Payload:
		// Ignore all messages created by the bot itself
		if evt.UserID == evt.Payload.MessageAuthor {
			return result
		}

		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MessageEventType

		// Determine if it's a direct message or group message
		// If GroupID is empty, it's typically a DM
		topicType := "group"
		if evt.Payload.GroupID == "" {
			result.DetailType = protocol.MessageDirectEvent
			topicType = "dm"
		} else {
			result.DetailType = protocol.MessageGroupEvent
			topicType = "group"
		}

		result.Data = protocol.MessageEventData{
			Self: protocol.Self{
				Platform: ID,
			},
			MessageId:  evt.Payload.MessageID,
			AltMessage: evt.Payload.MessagePlainContent,
			UserId:     evt.Payload.MessageAuthor,
			TopicId:    evt.Payload.ConverseID,
			TopicType:  topicType,
		}
	}

	return result
}
