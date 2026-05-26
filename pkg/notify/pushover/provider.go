// Package pushover implements Pushover notification provider.
package pushover

import (
	"fmt"
	"net/http"
	"time"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const ID = "pushover"

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
		"{schema}://{user_key}@{token}",
		"{schema}://{user_key}@{token}/{targets}",
	}
}

func (*plugin) Send(tokens types.KV, message notify.Message) error {
	return doSend(tokens, message, resty.New(), "https://api.pushover.net")
}

func doSend(tokens types.KV, message notify.Message, client *resty.Client, baseURL string) error {
	userKey, _ := tokens.String("user_key")
	token, _ := tokens.String("token")

	client.SetTimeout(time.Minute)

	resp, err := client.R().SetBody(map[string]any{
		"token":    token,
		"user":     userKey,
		"title":    message.Title,
		"message":  message.Body,
		"url":      message.Url,
		"priority": message.Priority,
	}).Post(baseURL + "/1/messages.json")
	if err != nil {
		flog.Error(fmt.Errorf("[pushover] send failed: %w", err))
		return fmt.Errorf("pushover: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		flog.Error(fmt.Errorf("[pushover] send failed: non-200 response %d", resp.StatusCode()))
		return fmt.Errorf("pushover: non-200 response %d", resp.StatusCode())
	}

	flog.Debug("[pushover] sent notification: %s", message.Title)
	return nil
}
