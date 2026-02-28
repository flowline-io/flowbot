package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type Action struct {
	session *discordgo.Session
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
	var content string
	for _, segment := range in.Content {
		switch segment.Type {
		case "text":
			if text, ok := segment.Data["text"].(string); ok {
				if content != "" {
					content += "\n"
				}
				content += text
			}
		case "url":
			if url, ok := segment.Data["url"].(string); ok {
				if content != "" {
					content += "\n"
				}
				content += url
			}
		}
	}

	_, err := a.session.ChannelMessageSend(in.Channel, content)
	if err != nil {
		return fmt.Errorf("failed to send message to %s, %w", in.Channel, err)
	}
	return nil
}

func (a *Action) UpdateMessage(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) DeleteMessage(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

// request takes in the StatusCode and Content from other functions to display to the user's discord.
type request struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content protocol.Message `json:"body"`
	// Channel is the channel that the message will be sent to.
	Channel string `json:"channel"`
}
