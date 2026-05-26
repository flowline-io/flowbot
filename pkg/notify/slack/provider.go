// Package slack implements the Slack notification provider.
package slack

import (
	"fmt"
	"net/http"
	"time"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const ID = "slack"

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
		"{schema}://{tokenA}/{tokenB}/{tokenC}",
		"{schema}://{botname}@{tokenA}/{tokenB}/{tokenC}",
	}
}

func (*plugin) Send(tokens types.KV, message notify.Message) error {
	return doSend(tokens, message, resty.New(), "https://hooks.slack.com")
}

func doSend(tokens types.KV, message notify.Message, client *resty.Client, baseURL string) error {
	botname, _ := tokens.String("botname")
	tokenA, _ := tokens.String("tokenA")
	tokenB, _ := tokens.String("tokenB")
	tokenC, _ := tokens.String("tokenC")
	flog.Info("[slack] botname=%s", botname)

	url := fmt.Sprintf("%s/services/%s/%s/%s", baseURL, tokenA, tokenB, tokenC)

	client.SetTimeout(time.Minute)

	resp, err := client.R().SetBody(map[string]any{
		"text": message.Title,
		"blocks": []map[string]any{
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": message.Body,
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": message.Url,
				},
			},
		},
	}).Post(url)
	if err != nil {
		flog.Error(fmt.Errorf("[slack] send failed: %w", err))
		return fmt.Errorf("slack: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		flog.Error(fmt.Errorf("[slack] send failed: non-200 response %d", resp.StatusCode()))
		return fmt.Errorf("slack: non-200 response %d", resp.StatusCode())
	}

	flog.Debug("[slack] sent notification: %s", message.Title)
	return nil
}
