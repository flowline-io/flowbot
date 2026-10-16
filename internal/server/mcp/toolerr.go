package mcp

import "github.com/flowline-io/flowbot/pkg/toolerr"

// ToolError is the machine-readable MCP tool failure stored at structuredContent.error.
type ToolError = toolerr.ToolError

// ToToolError maps err onto the MCP tool error object.
// Plain errors and ErrInternal become code internal with a fixed message.
// ErrInvalidArgument keeps the cause text from types.ClientSafeDomainMessage.
func ToToolError(err error) ToolError {
	return toolerr.ToToolError(err)
}
