package pushover

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/notify"
)

const ID = "ntfy"

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
		"{schema}://{user_key}@{token}",
		"{schema}://{user_key}@{token}/{targets}",
	}
}

func (n *plugin) Send(tokens types.KV, message notify.Message) error {
	return nil
}
