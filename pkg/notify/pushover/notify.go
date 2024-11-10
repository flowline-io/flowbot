package pushover

import (
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const ID = "pushover"

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
