package slack

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/slack-go/slack"
)

type Action struct {
	api *slack.Client
}

func (a *Action) GetLatestEvents(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetSupportedActions(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetStatus(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetVersion(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) SendMessage(req protocol.Request) protocol.Response {
	channel, _ := types.KV(req.Params).String("topic") // fixme
	message, _ := types.KV(req.Params).Any("message")
	content, ok := message.(protocol.Message)
	if !ok {
		return protocol.NewFailedResponse(protocol.ErrBadSegmentType.New("message type error"))
	}
	if len(content) == 0 {
		return protocol.NewSuccessResponse(nil)
	}
	err := a.makeRequest(&request{
		Channel: channel,
		Content: content,
	})
	if err != nil {
		flog.Error(fmt.Errorf("failed to send message to %s, %w", channel, err))
		return protocol.NewFailedResponse(protocol.ErrInternalHandler.New("send message error"))
	}

	return protocol.NewSuccessResponse(nil)
}

func (a *Action) GetUserInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) CreateChannel(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetChannelInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetChannelList(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) RegisterChannels(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) RegisterSlashCommands(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) makeRequest(in *request) error {
	var msgOptions []slack.MsgOption
	var textParts []string
	var blocks []slack.Block

	for _, segment := range in.Content {
		switch segment.Type {
		case "text":
			if text, ok := segment.Data["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case "url":
			if url, ok := segment.Data["url"].(string); ok {
				textParts = append(textParts, url)
			}
		case "mention":
			if userId, ok := segment.Data["user_id"].(string); ok {
				textParts = append(textParts, fmt.Sprintf("<@%s>", userId))
			}
		case "mention_all":
			textParts = append(textParts, "<!channel>")
		case "image":
			if fileId, ok := segment.Data["file_id"].(string); ok {
				// For images, we can use file sharing or image block
				// Using image block for better display
				blocks = append(blocks, slack.NewImageBlock(fileId, fileId, "", nil))
			}
		case "file", "video", "audio", "voice":
			// For file attachments, Slack requires file upload first
			// For now, we'll include the file_id in text
			if fileId, ok := segment.Data["file_id"].(string); ok {
				textParts = append(textParts, fmt.Sprintf("[%s: %s]", segment.Type, fileId))
			}
		case "location":
			if lat, ok := segment.Data["latitude"].(float64); ok {
				if lon, ok2 := segment.Data["longitude"].(float64); ok2 {
					title, _ := segment.Data["title"].(string)
					content, _ := segment.Data["content"].(string)
					locationText := fmt.Sprintf("📍 %s\nLat: %f, Lon: %f", title, lat, lon)
					if content != "" {
						locationText += "\n" + content
					}
					textParts = append(textParts, locationText)
				}
			}
		case "reply":
			// Slack doesn't have native reply support in simple messages
			// We can mention the user and include the message reference
			if userId, ok := segment.Data["user_id"].(string); ok {
				if msgId, ok2 := segment.Data["message_id"].(string); ok2 {
					textParts = append(textParts, fmt.Sprintf("Replying to <@%s> (msg: %s)", userId, msgId))
				}
			}
		}
	}

	// Combine all text parts
	if len(textParts) > 0 {
		msgOptions = append(msgOptions, slack.MsgOptionText(strings.Join(textParts, "\n"), false))
	}

	// Add blocks if any
	if len(blocks) > 0 {
		msgOptions = append(msgOptions, slack.MsgOptionBlocks(blocks...))
	}

	// If no content, return error
	if len(msgOptions) == 0 {
		return fmt.Errorf("no valid message content")
	}

	_, _, err := a.api.PostMessage(
		in.Channel,
		msgOptions...,
	)
	if err != nil {
		return fmt.Errorf("failed to send message to %s, %w", in.Channel, err)
	}
	return nil
}

// SlackRequest takes in the StatusCode and Content from other functions to display to the user's slack.
type request struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content protocol.Message `json:"body"`
	// Channel is the channel that the message will be sent to.
	Channel string `json:"channel"`
}
