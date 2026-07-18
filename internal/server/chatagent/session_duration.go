package chatagent

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/types"
)

// SumSessionRunDurationMs returns cumulative run milliseconds across all completed
// user turns in the session's active branch. Each run stores RunDurationMs on one
// assistant message.
func SumSessionRunDurationMs(ctx context.Context, sessionID string) (int64, error) {
	storage := NewDBStorage(sessionID, types.Uid(""), "")
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return 0, err
	}
	return sumRunDurationFromBranch(branch), nil
}

// SumSessionsRunDurationMs returns cumulative run milliseconds for each session's
// active branch. leafBySession maps session flag to its current leaf entry id.
// Sessions with an empty leaf, missing entries, or walk errors are omitted.
func SumSessionsRunDurationMs(ctx context.Context, leafBySession map[string]string) (map[string]int64, error) {
	if len(leafBySession) == 0 {
		return map[string]int64{}, nil
	}
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}

	sessionIDs := make([]string, 0, len(leafBySession))
	for sessionID, leafID := range leafBySession {
		if leafID == "" {
			continue
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if len(sessionIDs) == 0 {
		return map[string]int64{}, nil
	}

	rows, err := store.Database.ListChatSessionEntriesBySessions(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}

	bySession := make(map[string][]session.TreeEntry, len(sessionIDs))
	for _, row := range rows {
		entry, err := rowToTreeEntry(row)
		if err != nil {
			return nil, err
		}
		bySession[row.SessionID] = append(bySession[row.SessionID], entry)
	}

	out := make(map[string]int64, len(leafBySession))
	for sessionID, leafID := range leafBySession {
		if leafID == "" {
			continue
		}
		branch, err := walkBranchFromEntries(bySession[sessionID], leafID)
		if err != nil {
			continue
		}
		if total := sumRunDurationFromBranch(branch); total > 0 {
			out[sessionID] = total
		}
	}
	return out, nil
}

func sumRunDurationFromBranch(branch []session.TreeEntry) int64 {
	var total int64
	for _, entry := range branch {
		if entry.Type != session.EntryMessage || entry.Message == nil {
			continue
		}
		assistant, ok := entry.Message.(msg.AssistantMessage)
		if !ok || assistant.RunDurationMs <= 0 {
			continue
		}
		total += assistant.RunDurationMs
	}
	return total
}
