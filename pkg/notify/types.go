// Package notify provides the notification dispatcher interface and registry.
package notify

import "github.com/flowline-io/flowbot/pkg/types"

type Notifyer interface {
	// Protocol Define protocol
	Protocol() string
	// Templates Define object templates
	Templates() []string
	// Send notify
	Send(tokens types.KV, message Message) error
}

// ConnectivityTestTemplateID is the template ID written by web UI channel connectivity tests.
// Retry for these records re-runs the channel probe instead of GatewaySend.
const ConnectivityTestTemplateID = "test"

// IsConnectivityTestTemplate reports whether templateID identifies a connectivity-test record.
func IsConnectivityTestTemplate(templateID string) bool {
	return templateID == ConnectivityTestTemplateID
}

type Priority int32

const (
	Low Priority = iota + 1
	Moderate
	Normal
	High
	Emergency
)

type Message struct {
	Title    string
	Body     string
	Url      string
	Priority Priority
}
