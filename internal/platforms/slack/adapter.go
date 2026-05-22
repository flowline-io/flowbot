package slack

import (
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type Adapter struct{}

func (*Adapter) MessageConvert(data any) protocol.Message {
	return platforms.MessageConvert(data)
}

func (*Adapter) EventConvert(data any) protocol.Event {
	var result protocol.Event
	evt, ok := data.(socketmode.Event)
	if !ok {
		return result
	}

	switch evt.Type {
	case socketmode.EventTypeConnecting:
		flog.Debug("Connecting to Slack with Socket Mode...")
	case socketmode.EventTypeConnectionError:
		flog.Debug("Connection failed. Retrying later...")
	case socketmode.EventTypeConnected:
		flog.Debug("Connected to Slack with Socket Mode.")
	case socketmode.EventTypeHello:
		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MetaEventType
		result.DetailType = protocol.MetaConnectEvent
	case socketmode.EventTypeEventsAPI:
		result = convertEventsAPIEvent(evt.Data.(slackevents.EventsAPIEvent))
	case socketmode.EventTypeSlashCommand:
		cmd, ok := evt.Data.(slack.SlashCommand)
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
	case socketmode.EventTypeInteractive:
		callback, ok := evt.Data.(slack.InteractionCallback)
		if !ok {
			flog.Debug("Ignored interactive event: %+v", evt)
			return result
		}
		result = convertInteractiveEvent(&callback)
	}

	return result
}

func convertEventsAPIEvent(apiEvent slackevents.EventsAPIEvent) protocol.Event {
	var result protocol.Event

	if apiEvent.InnerEvent.Type != "message" {
		return result
	}

	messageEvent, ok := apiEvent.InnerEvent.Data.(*slackevents.MessageEvent)
	if !ok {
		return result
	}

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

	setThreadContext(messageEvent.Channel, messageEvent.ThreadTimeStamp)

	result.Data = protocol.MessageEventData{
		Self: protocol.Self{
			Platform: ID,
		},
		MessageId:  messageEvent.ClientMsgID,
		AltMessage: messageEvent.Text,
		UserId:     messageEvent.User,
		TopicId:    messageEvent.Channel,
		TopicType:  messageEvent.ChannelType,
	}

	return result
}

func convertInteractiveEvent(callback *slack.InteractionCallback) protocol.Event {
	var result protocol.Event

	switch callback.Type {
	case slack.InteractionTypeBlockActions:
		if len(callback.ActionCallback.BlockActions) > 0 {
			action := callback.ActionCallback.BlockActions[0]
			result.Id = types.Id()
			result.Time = time.Now().UnixMicro()
			result.Type = protocol.MessageEventType
			result.DetailType = protocol.MessageDirectEvent
			setThreadContext(callback.Channel.ID, callback.MessageTs)

			result.Data = protocol.MessageEventData{
				Self: protocol.Self{
					Platform: ID,
				},
				MessageId:  callback.MessageTs,
				AltMessage: action.Value,
				UserId:     callback.User.ID,
				TopicId:    callback.Channel.ID,
				TopicType:  "im",
				Option:     action.ActionID,
			}
			flog.Info("Block action: user=%s action_id=%s value=%s",
				callback.User.ID, action.ActionID, action.Value)
		}

	case slack.InteractionTypeViewSubmission:
		result.Id = types.Id()
		result.Time = time.Now().UnixMicro()
		result.Type = protocol.MessageEventType
		result.DetailType = protocol.MessageDirectEvent

		submittedValues := make(map[string]string)
		for blockID := range callback.View.State.Values {
			blockValues := callback.View.State.Values[blockID]
			for actionID := range blockValues {
				actionValue := blockValues[actionID]
				key := blockID
				if key == "" {
					key = actionID
				}
				submittedValues[key] = actionValue.Value
				if actionValue.SelectedOption.Value != "" {
					submittedValues[key] = actionValue.SelectedOption.Value
				}
				if actionValue.SelectedDate != "" {
					submittedValues[key] = actionValue.SelectedDate
				}
				if actionValue.SelectedTime != "" {
					submittedValues[key] = actionValue.SelectedTime
				}
			}
		}

		var formParts []string
		for k, v := range submittedValues {
			formParts = append(formParts, k+"="+v)
		}

		flog.Info("Modal submission: callback_id=%s user=%s values=%+v",
			callback.View.CallbackID, callback.User.ID, submittedValues)

		result.Data = protocol.MessageEventData{
			Self: protocol.Self{
				Platform: ID,
			},
			AltMessage: callback.View.CallbackID + "\n" + strings.Join(formParts, "\n"),
			UserId:     callback.User.ID,
			Option:     "form_submit",
		}

	case slack.InteractionTypeShortcut:
		flog.Debug("Shortcut interaction: %s", callback.CallbackID)

	case slack.InteractionTypeDialogSubmission:
		flog.Debug("Dialog submission: %s", callback.CallbackID)

	default:
		flog.Debug("Unhandled interactive type: %s", callback.Type)
	}

	return result
}
