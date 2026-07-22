package chatagent

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/types"
)

// HistoryMessage is one persisted chat row exposed to HTTP clients.
type HistoryMessage struct {
	Role               string    `json:"role"`
	Kind               string    `json:"kind"`
	Text               string    `json:"text"`
	CreatedAt          time.Time `json:"created_at"`
	ToolName           string    `json:"tool_name,omitempty"`
	ToolStatus         string    `json:"tool_status,omitempty"`
	DurationMs         int64     `json:"duration_ms,omitempty"`
	TurnDurationMs     int64     `json:"turn_duration_ms,omitempty"`
	ThinkingDurationMs int64     `json:"thinking_duration_ms,omitempty"`
	RunDurationMs      int64     `json:"run_duration_ms,omitempty"`
	ThinkingText       string    `json:"thinking_text,omitempty"`
}

// ListSessionMessages returns user and assistant messages for a session branch.
func ListSessionMessages(ctx context.Context, sessionID string) ([]HistoryMessage, error) {
	storage := NewDBStorage(sessionID, types.Uid(""), "")
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return nil, err
	}
	createdAtByID, err := entryCreatedAtMap(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]HistoryMessage, 0, len(branch))
	for _, entry := range branch {
		switch entry.Type {
		case session.EntryMessage:
			if entry.Message == nil {
				continue
			}
			createdAt := createdAtByID[entry.ID]
			rows := historyMessagesFromMessage(entry.Message, createdAt)
			messages = append(messages, rows...)
		case session.EntryCompaction:
			text := strings.TrimSpace(entry.Summary)
			if text == "" {
				continue
			}
			createdAt := createdAtByID[entry.ID]
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			messages = append(messages, HistoryMessage{
				Role:      "assistant",
				Kind:      "assistant",
				Text:      text,
				CreatedAt: createdAt,
			})
		}
	}
	return messages, nil
}

// HasPersistedToolResults reports whether history rows include persisted tool result messages.
func HasPersistedToolResults(messages []HistoryMessage) bool {
	for _, m := range messages {
		if m.Kind == "tool" {
			return true
		}
	}
	return false
}

func entryCreatedAtMap(ctx context.Context, sessionID string) (map[string]time.Time, error) {
	if store.Database == nil {
		return map[string]time.Time{}, nil
	}
	rows, err := store.Database.ListChatSessionEntries(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	createdAtByID := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		createdAtByID[row.Flag] = row.CreatedAt
	}
	return createdAtByID, nil
}

func historyMessagesFromMessage(message agent.AgentMessage, createdAt time.Time) []HistoryMessage {
	switch m := message.(type) {
	case msg.UserMessage:
		text := strings.TrimSpace(textFromParts(m.Parts))
		if text == "" {
			return nil
		}
		ts := messageTimestamp(m.Timestamp, createdAt)
		return []HistoryMessage{{
			Role:      "user",
			Kind:      "user",
			Text:      text,
			CreatedAt: ts,
		}}
	case msg.AssistantMessage:
		ts := messageTimestamp(m.Timestamp, createdAt)
		out := make([]HistoryMessage, 0, 2)
		if thinking := strings.TrimSpace(m.ThinkingText); thinking != "" {
			out = append(out, HistoryMessage{
				Role:               "assistant",
				Kind:               "thinking",
				Text:               thinking,
				ThinkingText:       thinking,
				ThinkingDurationMs: m.ThinkingDurationMs,
				CreatedAt:          ts,
			})
		}
		// Tool-call assistants must not use AssistantDisplayText here: that summary
		// is later classified as a completed tool card even before approval/result.
		if len(m.ToolCalls()) > 0 {
			text := strings.TrimSpace(msg.TrimToolCallStreamContent(m.TextContent()))
			if text != "" {
				out = append(out, HistoryMessage{
					Role:           "assistant",
					Kind:           "assistant",
					Text:           text,
					TurnDurationMs: m.TurnDurationMs,
					RunDurationMs:  m.RunDurationMs,
					CreatedAt:      ts,
				})
			}
			return out
		}
		text := strings.TrimSpace(msg.AssistantDisplayText(m))
		if text != "" {
			out = append(out, HistoryMessage{
				Role:           "assistant",
				Kind:           "assistant",
				Text:           text,
				TurnDurationMs: m.TurnDurationMs,
				RunDurationMs:  m.RunDurationMs,
				CreatedAt:      ts,
			})
		}
		return out
	case msg.ToolResultMessage:
		text := strings.TrimSpace(textFromParts(m.Parts))
		status := "completed"
		if m.IsError {
			status = "error"
		}
		ts := messageTimestamp(m.Timestamp, createdAt)
		return []HistoryMessage{{
			Role:       "tool",
			Kind:       "tool",
			Text:       text,
			ToolName:   m.Name,
			ToolStatus: status,
			DurationMs: m.DurationMs,
			CreatedAt:  ts,
		}}
	default:
		return nil
	}
}

func messageTimestamp(messageTime, fallback time.Time) time.Time {
	if !messageTime.IsZero() {
		return messageTime
	}
	return fallback
}
