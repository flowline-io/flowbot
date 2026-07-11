package session

import (
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
)

func payloadFromRaw(raw any) result.Result[map[string]any, result.ParseError] {
	payload, ok := raw.(map[string]any)
	if !ok {
		data, err := sonic.Marshal(raw)
		if err != nil {
			return result.Err[map[string]any, result.ParseError](
				result.NewParseError("invalid_payload", "marshal message payload", err),
			)
		}
		if err := sonic.Unmarshal(data, &payload); err != nil {
			return result.Err[map[string]any, result.ParseError](
				result.NewParseError("invalid_payload", "unmarshal message payload", err),
			)
		}
	}
	return result.Ok[map[string]any, result.ParseError](payload)
}

func assistantFromPayload(payload map[string]any, text string) msg.AgentMessage {
	modelName := optionalStringField(payload, "model")
	stopReason := optionalStringField(payload, "stop_reason")
	parts := []msg.ContentPart{msg.TextPart{Text: text}}
	if rawCalls, ok := payload["tool_calls"].([]any); ok {
		for _, rawCall := range rawCalls {
			callMap, ok := rawCall.(map[string]any)
			if !ok {
				continue
			}
			parts = append(parts, msg.ToolCallPart{
				ID:        optionalStringField(callMap, "id"),
				Name:      optionalStringField(callMap, "name"),
				Arguments: optionalStringField(callMap, "arguments"),
			})
		}
	}
	assistant := msg.AssistantMessage{Parts: parts, Model: modelName, StopReason: stopReason}
	if rawUsage, ok := payload["usage"].(map[string]any); ok {
		assistant.Usage = usageFromRaw(rawUsage)
	}
	assistant.TurnDurationMs = optionalIntField(payload, "turn_duration_ms")
	assistant.ThinkingDurationMs = optionalIntField(payload, "thinking_duration_ms")
	assistant.ThinkingText = optionalStringField(payload, "thinking_text")
	assistant.RunDurationMs = optionalIntField(payload, "run_duration_ms")
	return assistant
}

func toolResultFromPayload(payload map[string]any, text string) result.Result[msg.AgentMessage, result.ParseError] {
	toolCallID, err := stringField(payload, "tool_call_id")
	if err != nil {
		return result.Err[msg.AgentMessage, result.ParseError](parseFieldError("tool_call_id", err))
	}
	name, err := stringField(payload, "name")
	if err != nil {
		return result.Err[msg.AgentMessage, result.ParseError](parseFieldError("name", err))
	}
	isError, err := optionalBoolField(payload, "is_error")
	if err != nil {
		return result.Err[msg.AgentMessage, result.ParseError](parseFieldError("is_error", err))
	}
	return result.Ok[msg.AgentMessage, result.ParseError](msg.ToolResultMessage{
		ToolCallID: toolCallID,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    isError,
		DurationMs: optionalIntField(payload, "duration_ms"),
	})
}

func parseMessage(raw any) result.Result[msg.AgentMessage, result.ParseError] {
	payloadResult := payloadFromRaw(raw)
	if !payloadResult.IsOk() {
		return result.Err[msg.AgentMessage, result.ParseError](payloadResult.ErrorValue())
	}
	payload := payloadResult.Value()
	role, err := stringField(payload, "role")
	if err != nil {
		return result.Err[msg.AgentMessage, result.ParseError](parseFieldError("role", err))
	}
	text, err := stringField(payload, "text")
	if err != nil {
		return result.Err[msg.AgentMessage, result.ParseError](parseFieldError("text", err))
	}
	switch role {
	case "user":
		return result.Ok[msg.AgentMessage, result.ParseError](msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: text}}})
	case "assistant":
		return result.Ok[msg.AgentMessage, result.ParseError](assistantFromPayload(payload, text))
	case "toolResult":
		parsed := toolResultFromPayload(payload, text)
		if !parsed.IsOk() {
			return result.Err[msg.AgentMessage, result.ParseError](parsed.ErrorValue())
		}
		return parsed
	default:
		return result.Err[msg.AgentMessage, result.ParseError](
			result.NewParseError("unknown_role", fmt.Sprintf("unknown message role %q", role), nil),
		)
	}
}

func rawToMessage(raw any) (msg.AgentMessage, error) {
	return result.GetOrError(parseMessage(raw))
}

func parseFieldError(field string, err error) result.ParseError {
	return result.NewParseError("invalid_field", fmt.Sprintf("field %q: %v", field, err), err)
}

func optionalBoolField(payload map[string]any, key string) (bool, error) {
	raw, ok := payload[key]
	if !ok {
		return false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("invalid bool field %q", key)
	}
	return value, nil
}
