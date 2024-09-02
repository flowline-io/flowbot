package tify

import "github.com/flowline-io/flowbot/pkg/notify"

const ID = "tify"

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
		"{schema}://{host}/{token}",
		"{schema}://{host}:{port}/{token}",
		"{schema}://{host}{path}{token}",
		"{schema}://{host}:{port}{path}{token}",
	}
}

func (n *plugin) Send(message notify.Message) error {
	return nil
}
