package tailchat

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

type Action struct {
	client *client
}

type client struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	accessToken  string
}

func newClient() *client {
	v := &client{
		clientId:     config.App.Platform.Tailchat.AppID,
		clientSecret: config.App.Platform.Tailchat.AppSecret,
	}

	v.c = resty.New()
	v.c.SetBaseURL(config.App.Platform.Tailchat.ApiURL)
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *client) auth() error {
	resp, err := v.c.R().
		SetResult(&TokenResponse{}).
		SetBody(types.KV{
			"appId": v.clientId,
			"token": utils.MD5(v.clientId + v.clientSecret),
		}).
		Post("/api/openapi/bot/login")
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*TokenResponse)
		v.accessToken = result.Data.Jwt
		return nil
	} else {
		return fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *client) sendMessage(data SendMessageData) error {
	resp, err := v.c.R().
		SetBody(data).
		SetHeader("X-Token", v.accessToken).
		Post("/api/chat/message/sendMessage")
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
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

	// Ensure client is authenticated
	if a.client.accessToken == "" {
		err := a.client.auth()
		if err != nil {
			flog.Error(fmt.Errorf("failed to auth tailchat client: %w", err))
			return protocol.NewFailedResponse(protocol.ErrInternalHandler.New("auth error"))
		}
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

	err := a.client.sendMessage(SendMessageData{
		ConverseId: in.Channel,
		Content:    content,
		Plain:      content,
	})
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

// request takes in the StatusCode and Content from other functions to display to the user's tailchat.
type request struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content protocol.Message `json:"body"`
	// Channel is the channel that the message will be sent to.
	Channel string `json:"channel"`
}
