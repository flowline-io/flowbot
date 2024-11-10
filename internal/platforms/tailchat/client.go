package tailchat

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/go-resty/resty/v2"
)

type Payload struct {
	ID      string `json:"_id"`
	UserID  string `json:"userId"`
	Type    string `json:"type"`
	Payload struct {
		GroupID             string `json:"groupId"`
		ConverseID          string `json:"converseId"`
		MessageID           string `json:"messageId"`
		MessageAuthor       string `json:"messageAuthor"`
		MessageSnippet      string `json:"messageSnippet"`
		MessagePlainContent string `json:"messagePlainContent"`
	} `json:"payload"`
	Readed    bool      `json:"readed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	V         int       `json:"__v"`
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
		_, _ = fmt.Println(string(resp.Body()))
		return nil
	} else {
		return fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

type SendMessageData struct {
	ConverseId string          `json:"converseId"`
	GroupId    string          `json:"groupId"`
	Content    string          `json:"content"`
	Plain      string          `json:"plain"`
	Meta       SendMessageMeta `json:"meta"`
}

type SendMessageMeta struct {
	Mentions []string         `json:"mentions"`
	Reply    SendMessageReply `json:"reply"`
}

type SendMessageReply struct {
	Id      string `json:"_id"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

type TokenResponse struct {
	Data struct {
		Jwt string `json:"jwt"`
	} `json:"data"`
}
