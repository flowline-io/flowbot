package idempotency

import (
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
)

// MaxResultBytes is the success body cap shared by MCP and HTTP idempotency.
// It follows mcp.max_output_bytes and falls back to capability.DefaultMaxOutputBytes.
func MaxResultBytes() int {
	if config.App.MCP.MaxOutputBytes <= 0 {
		return capability.DefaultMaxOutputBytes
	}
	return config.App.MCP.MaxOutputBytes
}
