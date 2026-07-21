// Package notify provides the chatagent tool for Gateway notification pushes.
package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	pkgnotify "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// SendToolName is the agent tool name for sending a gateway notification.
	SendToolName = "send_notification"
)

// SendTool sends a notification via the notification gateway defaults.
type SendTool struct {
	// UID is the notification owner recorded with the send.
	UID types.Uid
}

// Name returns the tool identifier.
func (SendTool) Name() string { return SendToolName }

// Description explains the tool to the model.
func (SendTool) Description() string {
	return "Send a push notification through the configured default notification channel and template. Use for reminders and alerts outside the current chat."
}

// Parameters returns the JSON schema for tool arguments.
func (SendTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Notification body text (mapped to template summary)",
			},
		},
		"required": []string{"message"},
	}
}

// Execute sends the notification using global default channel and template.
func (t SendTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	message := strings.TrimSpace(fmt.Sprint(args["message"]))
	if message == "" || message == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "message is required", "pass the notification text"), nil
	}

	err := pkgnotify.GatewaySendDefaults(ctx, t.UID, map[string]any{
		pkgnotify.PayloadKeySummary: message,
	})
	if err != nil {
		return sendErrorResult(id, t.Name(), err), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: "notification sent"}},
	}, nil
}

// Register registers send_notification on the given registry.
func Register(registry *tool.Registry, uid types.Uid) error {
	if registry == nil {
		return fmt.Errorf("notify tools: registry is nil")
	}
	return registry.Register(SendTool{UID: uid})
}

// ActiveToolNames returns the default notify tool names.
func ActiveToolNames() []string {
	return []string{SendToolName}
}

func sendErrorResult(callID, name string, err error) msg.ToolResultMessage {
	code := "tool_error"
	hint := "retry or check notification gateway configuration"
	switch {
	case errors.Is(err, pkgnotify.ErrNoDefaultChannel):
		code = "unavailable"
		hint = "set a default notification channel in Notifications settings"
	case errors.Is(err, pkgnotify.ErrNoDefaultTemplate):
		code = "unavailable"
		hint = "set a default notification template in Notifications settings"
	case errors.Is(err, types.ErrUnavailable):
		code = "unavailable"
		hint = "notification store is not available"
	case errors.Is(err, types.ErrInvalidArgument):
		code = "invalid_args"
		hint = "fix the tool arguments"
	case errors.Is(err, types.ErrNotFound):
		code = "not_found"
		hint = "check default template and channel still exist"
	}
	return tool.ErrorResult(callID, name, code, err.Error(), hint)
}
