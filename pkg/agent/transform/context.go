package transform

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// Context applies the default no-op context transform.
func Context(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
	return FilterContext(messages)
}
