package chatagent

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// IsChatControlCommand reports whether the message is a chat session control command.
func IsChatControlCommand(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "chat", "end", "help":
		return true
	default:
		return false
	}
}

// CreateSession persists a new chat session row for the user.
func CreateSession(ctx context.Context, uid types.Uid, sessionID string) error {
	now := time.Now().UTC()
	return store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag:      sessionID,
		UID:       uid.String(),
		LeafID:    "",
		State:     int(schema.ChatSessionActive),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

// CloseSession marks a chat session as closed without deleting history.
func CloseSession(ctx context.Context, sessionID string) error {
	return store.Database.CloseChatSession(ctx, sessionID)
}
