// Package ntfy implements the ntfy notification provider.
package ntfy

import (
	"fmt"
	"net/http"
	"time"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const ID = "ntfy"

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
		"{schema}://{topic}",
		"{schema}://{host}/{targets}",
		"{schema}://{host}:{port}/{targets}",
		"{schema}://{user}@{host}/{targets}",
		"{schema}://{user}@{host}:{port}/{targets}",
		"{schema}://{user}:{password}@{host}/{targets}",
		"{schema}://{user}:{password}@{host}:{port}/{targets}",
		"{schema}://{token}@{host}/{targets}",
		"{schema}://{token}@{host}:{port}/{targets}",
	}
}

func (*plugin) Send(tokens types.KV, message notify.Message) error {
	host, _ := tokens.String("host")
	return doSend(tokens, message, resty.New(), fmt.Sprintf("http://%s", host))
}

func doSend(tokens types.KV, message notify.Message, client *resty.Client, baseURL string) error {
	topic, _ := tokens.String("topic")

	client.SetBaseURL(baseURL)
	client.SetTimeout(time.Minute)

	resp, err := client.R().SetBody(map[string]any{
		"topic":    topic,
		"title":    message.Title,
		"message":  message.Body,
		"priority": message.Priority,
	}).Post("/")
	if err != nil {
		flog.Error(fmt.Errorf("[ntfy] send failed: %w", err))
		return fmt.Errorf("ntfy: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		flog.Error(fmt.Errorf("[ntfy] send failed: non-200 response %d", resp.StatusCode()))
		return fmt.Errorf("ntfy: non-200 response %d", resp.StatusCode())
	}

	flog.Debug("[ntfy] sent notification: %s", message.Title)
	return nil
}
