package ntfy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/go-resty/resty/v2"
)

const ID = "ntfy"

var handler plugin

type plugin struct{}

func Register() {
	notify.Register(ID, &handler)
}

func (n *plugin) Protocol() string {
	return ID
}

func (n *plugin) Templates() []string {
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

func (n *plugin) Send(tokens types.KV, message notify.Message) error {
	host, _ := tokens.String("host")
	topic, _ := tokens.String("topic")
	url := fmt.Sprintf("http://%s", host)

	c := resty.New()
	c.SetBaseURL(url)
	c.SetTimeout(time.Minute)

	resp, err := c.R().SetBody(map[string]interface{}{
		"topic":    topic,
		"title":    message.Title,
		"message":  message.Body,
		"priority": message.Priority,
	}).Post("/")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("%d", resp.StatusCode())
	}

	return nil
}
