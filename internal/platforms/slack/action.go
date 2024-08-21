package slack

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/slack-go/slack"
)

type Action struct {
	api *slack.Client
}

func (a *Action) GetLatestEvents(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetSupportedActions(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetStatus(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetVersion(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) SendMessage(req protocol.Request) protocol.Response {
	channel, _ := types.KV(req.Params).String("topic") // fixme
	message, _ := types.KV(req.Params).Any("message")
	content, ok := message.(protocol.Message)
	if !ok {
		return protocol.NewFailedResponse(protocol.ErrBadSegmentType)
	}
	err := a.makeRequest(&request{
		Channel: channel,
		Content: content,
	})
	if err != nil {
		flog.Error(fmt.Errorf("failed to send message to %s, %w", channel, err))
		return protocol.NewFailedResponse(protocol.ErrInternalHandler)
	}

	return protocol.NewSuccessResponse(nil)
}

func (a *Action) GetUserInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) CreateChannel(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelList(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterChannels(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterSlashCommands(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) makeRequest(in *request) error {
	var msgOptions []slack.MsgOption
	for _, segment := range in.Content {
		switch segment.Type {
		case "text":
			msgOptions = append(msgOptions, slack.MsgOptionText(segment.Data["text"].(string), false))
		case "url":
			msgOptions = append(msgOptions, slack.MsgOptionText(segment.Data["url"].(string), false))
		}
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
