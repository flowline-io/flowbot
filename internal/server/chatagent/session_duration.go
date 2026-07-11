package chatagent

import (
	"context"

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
