package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AbilityInvoker invokes a capability operation; defaults to capability.Invoke.
type AbilityInvoker func(ctx context.Context, capType hub.CapabilityType, operation string, params map[string]any) (*capability.InvokeResult, error)

// AbilityTool exposes a readonly ability operation as an agent tool.
type AbilityTool struct {
	cfg    config.AbilityToolConfig
	invoke AbilityInvoker
}

// NewAbilityTool builds an AbilityTool from config. Readonly must be true.
func NewAbilityTool(cfg config.AbilityToolConfig, invoke AbilityInvoker) (*AbilityTool, error) {
	if err := ValidateAbilityToolConfig(cfg); err != nil {
		return nil, err
	}
	if invoke == nil {
		invoke = capability.Invoke
	}
	return &AbilityTool{cfg: cfg, invoke: invoke}, nil
}

// ValidateAbilityToolConfig rejects non-readonly or incomplete ability tool entries.
func ValidateAbilityToolConfig(cfg config.AbilityToolConfig) error {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return fmt.Errorf("ability_tools: name is required")
	}
	if strings.TrimSpace(cfg.Capability) == "" {
		return fmt.Errorf("ability_tools[%s]: capability is required", name)
	}
	if strings.TrimSpace(cfg.Operation) == "" {
		return fmt.Errorf("ability_tools[%s]: operation is required", name)
	}
	if !cfg.Readonly {
		return fmt.Errorf("ability_tools[%s]: readonly must be true", name)
	}
	return nil
}

// Name returns the tool identifier.
func (t AbilityTool) Name() string { return strings.TrimSpace(t.cfg.Name) }

// Description explains the tool to the model.
func (t AbilityTool) Description() string {
	if desc := strings.TrimSpace(t.cfg.Description); desc != "" {
		return desc
	}
	return fmt.Sprintf("Readonly ability call %s.%s", t.cfg.Capability, t.cfg.Operation)
}

// Parameters returns a generic object schema for ability params.
func (AbilityTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"params": map[string]any{
				"type":        "object",
				"description": "Parameters forwarded to the ability operation",
			},
		},
	}
}

// Execute invokes the configured ability operation.
func (t AbilityTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	params := abilityParams(args)
	result, err := t.invoke(ctx, hub.CapabilityType(t.cfg.Capability), t.cfg.Operation, params)
	if err != nil {
		return abilityErrorResult(id, t.Name(), err), nil
	}
	text, err := formatAbilityResult(result)
	if err != nil {
		return tool.ErrorResult(id, t.Name(), "serialize_error", "failed to format ability result", "retry with simpler parameters"), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

// RegisterAbilityTools validates and registers configured ability tools.
func RegisterAbilityTools(registry *tool.Registry, entries []config.AbilityToolConfig, invoke AbilityInvoker) ([]string, error) {
	if registry == nil {
		return nil, fmt.Errorf("ability_tools: registry is required")
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		item, err := NewAbilityTool(entry, invoke)
		if err != nil {
			return nil, err
		}
		if err := registry.Register(item); err != nil {
			return nil, err
		}
		names = append(names, item.Name())
	}
	return names, nil
}

func abilityParams(args map[string]any) map[string]any {
	if args == nil {
		return map[string]any{}
	}
	if nested, ok := args["params"].(map[string]any); ok && nested != nil {
		return nested
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if k == "params" {
			continue
		}
		out[k] = v
	}
	return out
}

func formatAbilityResult(result *capability.InvokeResult) (string, error) {
	if result == nil {
		return "ok", nil
	}
	if text := strings.TrimSpace(result.Text); text != "" {
		return text, nil
	}
	if result.Data == nil {
		return "ok", nil
	}
	data, err := sonic.Marshal(result.Data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func abilityErrorResult(id, name string, err error) msg.ToolResultMessage {
	code, message, hint := mapAbilityError(err)
	return tool.ErrorResult(id, name, code, message, hint)
}

func mapAbilityError(err error) (code, message, hint string) {
	if err == nil {
		return "ability_error", "unknown ability error", "retry the request"
	}
	switch {
	case errors.Is(err, types.ErrNotFound):
		return "not_found", "resource not found", "check identifiers and retry"
	case errors.Is(err, types.ErrInvalidArgument):
		return "invalid_args", "invalid ability arguments", "fix parameters and retry"
	case errors.Is(err, types.ErrForbidden), errors.Is(err, types.ErrUnauthorized):
		return "forbidden", "ability access denied", "check credentials or permissions"
	case errors.Is(err, types.ErrTimeout):
		return "timeout", "ability call timed out", "retry later"
	case errors.Is(err, types.ErrNotImplemented):
		return "not_implemented", "ability operation is not available", "use a different operation"
	case errors.Is(err, types.ErrProvider), errors.Is(err, types.ErrUnavailable):
		return "unavailable", "ability backend unavailable", "retry later"
	default:
		// Avoid leaking provider raw errors to the model.
		return "ability_error", "ability call failed", "fix parameters or retry later"
	}
}
