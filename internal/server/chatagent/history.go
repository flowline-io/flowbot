package chatagent

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

// HistoryMessage is one persisted chat turn exposed to HTTP clients.
type HistoryMessage struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ListSessionMessages returns user and assistant messages for a session branch.
func ListSessionMessages(ctx context.Context, sessionID string) ([]HistoryMessage, error) {
	storage := NewDBStorage(sessionID)
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return nil, err
	}

	messages := make([]HistoryMessage, 0, len(branch))
	for _, entry := range branch {
		if entry.Type != session.EntryMessage || entry.Message == nil {
			continue
		}
		hm, ok := historyFromMessage(entry.Message)
		if !ok {
			continue
		}
		messages = append(messages, hm)
	}
	return messages, nil
}

func historyFromMessage(message agent.AgentMessage) (HistoryMessage, bool) {
	switch m := message.(type) {
	case msg.UserMessage:
		text := strings.TrimSpace(textFromParts(m.Parts))
		if text == "" {
			return HistoryMessage{}, false
		}
		return HistoryMessage{Role: "user", Text: text, CreatedAt: m.Timestamp}, true
	case msg.AssistantMessage:
		text := strings.TrimSpace(msg.AssistantDisplayText(m))
		if text == "" {
			return HistoryMessage{}, false
		}
		return HistoryMessage{Role: "assistant", Text: text, CreatedAt: m.Timestamp}, true
	default:
		return HistoryMessage{}, false
	}
}
