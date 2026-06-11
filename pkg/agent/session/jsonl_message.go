package session

import (
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

func payloadFromRaw(raw any) (map[string]any, error) {
	payload, ok := raw.(map[string]any)
	if !ok {
		data, err := sonic.Marshal(raw)
		if err != nil {
			return nil, err
		}
		if err := sonic.Unmarshal(data, &payload); err != nil {
			return nil, err
		}
	}
	return payload, nil
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
	return assistant
}

func toolResultFromPayload(payload map[string]any, text string) (msg.AgentMessage, error) {
	toolCallID, err := stringField(payload, "tool_call_id")
	if err != nil {
		return nil, err
	}
	name, err := stringField(payload, "name")
	if err != nil {
		return nil, err
	}
	isError, _ := boolField(payload, "is_error")
	return msg.ToolResultMessage{
		ToolCallID: toolCallID,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    isError,
	}, nil
}

func rawToMessage(raw any) (msg.AgentMessage, error) {
	payload, err := payloadFromRaw(raw)
	if err != nil {
		return nil, err
	}
	role, err := stringField(payload, "role")
	if err != nil {
		return nil, err
	}
	text, err := stringField(payload, "text")
	if err != nil {
		return nil, err
	}
	switch role {
	case "user":
		return msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: text}}}, nil
	case "assistant":
		return assistantFromPayload(payload, text), nil
	case "toolResult":
		return toolResultFromPayload(payload, text)
	default:
		return nil, fmt.Errorf("session jsonl: unknown message role %q", role)
	}
}
