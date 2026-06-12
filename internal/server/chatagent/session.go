package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
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
	err := store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag:      sessionID,
		UID:       uid.String(),
		LeafID:    "",
		State:     int(schema.ChatSessionActive),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] create session uid=%s session=%s: %w", uid, sessionID, err))
		return err
	}
	flog.Debug("[chat-agent] session row created uid=%s session=%s", uid, sessionID)
	return nil
}

// CloseSession marks a chat session as closed, cancels in-flight runs, and releases locks.
// The ordering (cancel -> close DB -> release lock) ensures no new run can start on a closing session.
func CloseSession(ctx context.Context, sessionID string) error {
	cancelRun(sessionID)
	EvictHarnessPool(sessionID)
	if err := store.Database.CloseChatSession(ctx, sessionID); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] close session session=%s: %w", sessionID, err))
		return err
	}
	releaseSessionLock(sessionID)
	flog.Debug("[chat-agent] session row closed session=%s", sessionID)
	return nil
}
