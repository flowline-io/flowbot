package ntfy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/go-resty/resty/v2"
)

const ID = "ntfy"

var handler plugin

type plugin struct {
	tokens types.KV
}

func init() {
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

func (n *plugin) ParseTokens(line string) error {
	kv, err := notify.ParseTemplate(line, n.Templates())
	if err != nil {
		return err
	}
	n.tokens = kv
	return nil
}

func (n *plugin) Send(message notify.Message) error {
	host, _ := n.tokens.String("host")
	topic, _ := n.tokens.String("topic")
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
