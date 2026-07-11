package chatagent

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/types"
)

// SessionExport is the full session snapshot returned by the export API.
type SessionExport struct {
	SessionID  string           `json:"session_id"`
	UID        string           `json:"uid"`
	LeafID     string           `json:"leaf_id"`
	State      string           `json:"state"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	ExportedAt time.Time        `json:"exported_at"`
	EntryCount int              `json:"entry_count"`
	Entries    []map[string]any `json:"entries"`
}

// ExportSession loads session metadata and all persisted tree entries.
func ExportSession(ctx context.Context, sessionID string) (*SessionExport, error) {
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	storage := NewDBStorage(sessionID, types.Uid(""), "")
	entries, err := storage.ListEntries(ctx)
	if err != nil {
		return nil, err
	}

	wireEntries, err := marshalExportEntries(entries)
	if err != nil {
		return nil, fmt.Errorf("chatagent export: marshal entries: %w", err)
	}

	return &SessionExport{
		SessionID:  sessionID,
		UID:        sess.UID,
		LeafID:     sess.LeafID,
		State:      sessionStateLabel(sess.State),
		CreatedAt:  sess.CreatedAt,
		UpdatedAt:  sess.UpdatedAt,
		ExportedAt: time.Now().UTC(),
		EntryCount: len(wireEntries),
		Entries:    wireEntries,
	}, nil
}

func marshalExportEntries(entries []session.TreeEntry) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		data, err := session.MarshalEntry(entry)
		if err != nil {
			return nil, err
		}
		var payload map[string]any
		if err := sonic.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("decode entry: %w", err)
		}
		out = append(out, payload)
	}
	return out, nil
}

func sessionStateLabel(state int) string {
	switch schema.ChatSessionState(state) {
	case schema.ChatSessionActive:
		return "active"
	case schema.ChatSessionClosed:
		return "closed"
	default:
		return "unknown"
	}
}
