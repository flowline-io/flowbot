package session

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// MarshalEntry serializes one session tree entry to JSON.
func MarshalEntry(entry TreeEntry) ([]byte, error) {
	payload := entry
	if entry.Message != nil {
		payload.MessageRaw = messageToRaw(entry.Message)
	}
	return sonic.Marshal(payload)
}

// UnmarshalEntry deserializes one session tree entry from JSON.
func UnmarshalEntry(data []byte) (TreeEntry, error) {
	var entry TreeEntry
	if err := sonic.Unmarshal(data, &entry); err != nil {
		return TreeEntry{}, fmt.Errorf("session jsonl: unmarshal entry: %w", err)
	}
	if entry.MessageRaw != nil {
		message, err := rawToMessage(entry.MessageRaw)
		if err != nil {
			return TreeEntry{}, err
		}
		entry.Message = message
	}
	return entry, nil
}

// SerializeSession writes session entries as JSONL.
func SerializeSession(entries []TreeEntry) ([]byte, error) {
	var buf bytes.Buffer
	for _, entry := range entries {
		line, err := MarshalEntry(entry)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(line); err != nil {
			return nil, err
		}
		if err := buf.WriteByte('\n'); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// DeserializeSession reads JSONL session entries.
func DeserializeSession(data []byte) ([]TreeEntry, error) {
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	entries := make([]TreeEntry, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		entry, err := UnmarshalEntry(line)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// LoadJSONL deserializes session entries from a reader.
func LoadJSONL(r io.Reader) ([]TreeEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("session jsonl: read: %w", err)
	}
	return DeserializeSession(data)
}

func messageToRaw(message msg.AgentMessage) map[string]any {
	switch m := message.(type) {
	case msg.UserMessage:
		return map[string]any{"role": "user", "text": textFromParts(m.Parts)}
	case msg.AssistantMessage:
		return map[string]any{"role": "assistant", "text": textFromParts(m.Parts), "model": m.Model}
	case msg.ToolResultMessage:
		return map[string]any{
			"role":         "toolResult",
			"tool_call_id": m.ToolCallID,
			"name":         m.Name,
			"text":         textFromParts(m.Parts),
			"is_error":     m.IsError,
		}
	default:
		return map[string]any{"role": string(message.Role())}
	}
}

func textFromParts(parts []msg.ContentPart) string {
	var text strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			text.WriteString(tp.Text)
		}
	}
	return text.String()
}

func stringField(payload map[string]any, key string) (string, error) {
	value, ok := payload[key].(string)
	if !ok {
		return "", fmt.Errorf("session jsonl: missing string field %q", key)
	}
	return value, nil
}

func boolField(payload map[string]any, key string) (bool, error) {
	value, ok := payload[key].(bool)
	if !ok {
		return false, fmt.Errorf("session jsonl: missing bool field %q", key)
	}
	return value, nil
}

func rawToMessage(raw any) (msg.AgentMessage, error) {
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
		modelName, err := stringField(payload, "model")
		if err != nil {
			return nil, err
		}
		return msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: text}}, Model: modelName}, nil
	case "toolResult":
		toolCallID, err := stringField(payload, "tool_call_id")
		if err != nil {
			return nil, err
		}
		name, err := stringField(payload, "name")
		if err != nil {
			return nil, err
		}
		isError, err := boolField(payload, "is_error")
		if err != nil {
			return nil, err
		}
		return msg.ToolResultMessage{
			ToolCallID: toolCallID,
			Name:       name,
			Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
			IsError:    isError,
		}, nil
	default:
		return nil, fmt.Errorf("session jsonl: unknown message role %q", role)
	}
}
