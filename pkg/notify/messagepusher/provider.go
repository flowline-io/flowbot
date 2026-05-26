// Package messagepusher implements the Message Pusher notification provider.
package messagepusher

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
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

	if domain == "" {
		domain = net.JoinHostPort(host, port)
	}
	baseURL := fmt.Sprintf("http://%s/push/%s", domain, user)

	return doSend(tokens, message, resty.New(), baseURL)
}

func doSend(tokens types.KV, message notify.Message, client *resty.Client, baseURL string) error {
	channel, _ := tokens.String("channel")
	token, _ := tokens.String("token")

	client.SetBaseURL(baseURL)
	client.SetTimeout(time.Minute)

	resp, err := client.R().SetQueryParams(map[string]string{
		"channel":     channel,
		"token":       token,
		"title":       message.Title,
		"description": message.Body,
		"url":         message.Url,
	}).SetResult(&Response{}).Get("/")
	if err != nil {
		flog.Error(fmt.Errorf("[message-pusher] send failed: %w", err))
		return fmt.Errorf("message-pusher: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		flog.Error(fmt.Errorf("[message-pusher] send failed: non-200 response %d", resp.StatusCode()))
		return fmt.Errorf("message-pusher: non-200 response %d", resp.StatusCode())
	}

	respResult, ok := resp.Result().(*Response)
	if !ok || !respResult.Success {
		msg := ""
		if respResult != nil {
			msg = respResult.Message
		}
		flog.Error(fmt.Errorf("[message-pusher] send failed: %s", msg))
		return fmt.Errorf("message-pusher: %s", msg)
	}

	flog.Debug("[message-pusher] sent notification: %s", message.Title)
	return nil
}

type Response struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
