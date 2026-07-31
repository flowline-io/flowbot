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
	return "Send a notification to the user's inbox (and optional external channels). Use for reminders and alerts outside the current chat. Optional url deep-links the inbox item."
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
			"title": map[string]any{
				"type":        "string",
				"description": "Optional notification title",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "Optional deep-link URL shown in the inbox",
			},
			"channels": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional channel names; default is inapp plus the configured default external channel",
			},
		},
		"required": []string{"message"},
	}
}

// Execute sends the notification using inbox defaults (inapp ± external).
func (t SendTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	message := strings.TrimSpace(fmt.Sprint(args["message"]))
	if message == "" || message == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "message is required", "pass the notification text"), nil
	}

	payload := map[string]any{
		pkgnotify.PayloadKeySummary: message,
	}
	if title := optionalStringArg(args, "title"); title != "" {
		payload[pkgnotify.PayloadKeyTitle] = title
	}
	if url := optionalStringArg(args, "url"); url != "" {
		payload[pkgnotify.PayloadKeyURL] = url
	}

	templateID, err := pkgnotify.ResolveDefaultTemplateID(ctx)
	if err != nil {
		if errors.Is(err, pkgnotify.ErrNoDefaultTemplate) {
			templateID = pkgnotify.AgentNotifyTemplateID
		} else {
			return sendErrorResult(id, t.Name(), err), nil
		}
	}

	channels := pkgnotify.DefaultInboxChannels(ctx)
	if raw, ok := args["channels"]; ok {
		if parsed, ok := parseStringSliceArg(raw); ok && len(parsed) > 0 {
			channels = parsed
		}
	}

	err = pkgnotify.GatewaySend(ctx, t.UID, templateID, channels, payload)
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

func optionalStringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func parseStringSliceArg(raw any) ([]string, bool) {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out, true
	default:
		return nil, false
	}
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
