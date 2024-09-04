package pushover

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/notify"
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
		"{schema}://{user_key}@{token}",
		"{schema}://{user_key}@{token}/{targets}",
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
	return nil
}
