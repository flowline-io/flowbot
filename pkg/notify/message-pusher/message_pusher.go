package message_pusher

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const ID = "message-pusher"

var handler plugin

type plugin struct{}

func Register() {
	notify.Register(ID, &handler)
}

func (*plugin) Protocol() string {
	return ID
}

func (*plugin) Templates() []string {
	return []string{
		"{schema}://{user}@{domain}/{channel}/{token}",
		"{schema}://{user}@{host}:{port}/{channel}/{token}",
	}
}

func (*plugin) Send(tokens types.KV, message notify.Message) error {
	user, _ := tokens.String("user")
	domain, _ := tokens.String("domain")
	host, _ := tokens.String("host")
	port, _ := tokens.String("port")
	channel, _ := tokens.String("channel")
	token, _ := tokens.String("token")

	if domain == "" {
		domain = net.JoinHostPort(host, port)
	}
	url := fmt.Sprintf("http://%s/push/%s", domain, user)

	c := resty.New()
	c.SetBaseURL(url)
	c.SetTimeout(time.Minute)

	resp, err := c.R().SetQueryParams(map[string]string{
		"channel":     channel,
		"token":       token,
		"title":       message.Title,
		"description": message.Body,
		"url":         message.Url,
	}).SetResult(&Response{}).Get("/")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("%d", resp.StatusCode())
	}

	respResult, ok := resp.Result().(*Response)
	if !ok || !respResult.Success {
		return fmt.Errorf("%s", respResult.Message)
	}

	return nil
}

type Response struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
