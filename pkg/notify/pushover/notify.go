package pushover

import (
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
	return nil
}
