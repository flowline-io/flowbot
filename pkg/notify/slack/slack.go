package slack

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/go-resty/resty/v2"
	"net/http"
	"time"
)

const ID = "slack"

var handler plugin

type plugin struct{}

func init() {
	notify.Register(ID, &handler)
}

func (n *plugin) Protocol() string {
	return ID
}

func (n *plugin) Templates() []string {
	return []string{
		"{schema}://{tokenA}/{tokenB}/{tokenC}",
		"{schema}://{botname}@{tokenA}/{tokenB}/{tokenC}",
	}
}

func (n *plugin) Send(tokens types.KV, message notify.Message) error {
	botname, _ := tokens.String("botname")
	tokenA, _ := tokens.String("tokenA")
	tokenB, _ := tokens.String("tokenB")
	tokenC, _ := tokens.String("tokenC")
	flog.Info("[slack] botname=%s", botname)

	url := fmt.Sprintf("https://hooks.slack.com/services/%s/%s/%s", tokenA, tokenB, tokenC)

	c := resty.New()
	c.SetTimeout(time.Minute)

	resp, err := c.R().SetBody(map[string]interface{}{
		"text": message.Title,
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": message.Body,
				},
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": message.Url,
				},
			},
		},
	}).Post(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("%d", resp.StatusCode())
	}

	return nil
}
