package tool

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// FormatToolError builds an actionable tool error message for the model.
func FormatToolError(code, message, hint string) string {
	if code == "" {
		code = "tool_error"
	}
	if hint == "" {
		return fmt.Sprintf("[%s] %s", code, message)
	}
	return fmt.Sprintf("[%s] %s. Hint: %s", code, message, hint)
}

// ErrorResult builds an inline tool error result with an actionable message.
func ErrorResult(callID, name, code, message, hint string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: callID,
		Name:       name,
		IsError:    true,
		Parts:      []msg.ContentPart{msg.TextPart{Text: FormatToolError(code, message, hint)}},
	}
}
