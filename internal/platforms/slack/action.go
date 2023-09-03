package slack

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/slack-go/slack"
	"strconv"
	"time"
)

type Action struct {
	api *slack.Client
}

func (a *Action) GetLatestEvents(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetSupportedActions(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetStatus(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetVersion(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) SendMessage(req protocol.Request) protocol.Response {
	text, _ := types.KV(req.Params).String("text")     // fixme
	channel, _ := types.KV(req.Params).String("topic") // fixme
	err := a.makeRequest(&SlackRequest{
		Channel: channel,
		Content: text,
	})
	if err != nil {
		return protocol.NewFailedResponse(protocol.ErrInternalHandler)
	}

	return protocol.NewSuccessResponse(nil)
}

func (a *Action) GetUserInfo(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) CreateChannel(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelInfo(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelList(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterChannels(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterSlashCommands(req protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) makeRequest(in *SlackRequest) error {
	code := strconv.Itoa(in.StatusCode)
	attachment := slack.Attachment{
		Color: "#0069ff",
		Fields: []slack.AttachmentField{
			{
				Title: in.Content,
				Value: fmt.Sprintf("Response: %s", code),
			},
		},
		Footer: "FlowBot " + " | " + time.Now().Format("01-02-2006 3:4:5 MST"),
	}
	_, _, err := a.api.PostMessage(
		in.Channel,
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return err
	}
	return nil
}

// SlackRequest takes in the StatusCode and Content from other functions to display to the user's slack.
type SlackRequest struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content string `json:"body"`
	// Channel is the channel that the message will be sent to.
	Channel string `json:"channel"`
}
