// Package inapp implements the in-app inbox notification provider.
package inapp

import (
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// ID is the notify protocol and seeded channel name.
	ID = "inapp"
)

var handler plugin

type plugin struct{}

// Register registers the inapp notifyer.
func Register() {
	notify.Register(ID, &handler)
}

func (*plugin) Protocol() string {
	return ID
}

func (*plugin) Templates() []string {
	return []string{
		"{schema}://inbox",
	}
}

// Send is a no-op: delivery is the notification_records row written by the gateway.
func (*plugin) Send(_ types.KV, _ notify.Message) error {
	return nil
}
