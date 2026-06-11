package session

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

func applyPathMetadata(path []TreeEntry, ctx *Context) *TreeEntry {
	var compaction *TreeEntry
	for i := range path {
		entry := &path[i]
		switch entry.Type {
		case EntryModelChange:
			ctx.ModelName = entry.ModelName
		case EntryActiveToolsChange:
			ctx.ActiveTools = append([]string(nil), entry.ActiveToolNames...)
		case EntryCompaction:
			compaction = entry
		}
	}
	return compaction
}

func appendContextMessage(ctx *Context, entry TreeEntry) {
	switch entry.Type {
	case EntryMessage:
		if entry.Message != nil {
			ctx.Messages = append(ctx.Messages, entry.Message)
		}
	case EntryBranchSummary:
		ctx.Messages = append(ctx.Messages, msg.BranchSummaryMessage{
			Summary:   entry.Summary,
			FromID:    entry.FromID,
			Timestamp: time.Now().UTC(),
		})
	}
}

func appendCompactionContext(ctx *Context, path []TreeEntry, compaction *TreeEntry) {
	ctx.Messages = append(ctx.Messages, msg.CompactionSummaryMessage{
		Summary:      compaction.Summary,
		TokensBefore: compaction.TokensBefore,
		Timestamp:    time.Now().UTC(),
	})

	compactionIdx := indexOfEntry(path, compaction.ID)
	foundFirstKept := false
	if compaction.FirstKeptEntryID != "" && indexOfEntry(path, compaction.FirstKeptEntryID) < 0 {
		// Invalid compaction metadata: fall back to post-compaction messages only.
		foundFirstKept = false
	}
	for i := range compactionIdx {
		entry := path[i]
		if entry.ID == compaction.FirstKeptEntryID {
			foundFirstKept = true
		}
		if foundFirstKept {
			appendContextMessage(ctx, entry)
		}
	}
	for i := compactionIdx + 1; i < len(path); i++ {
		appendContextMessage(ctx, path[i])
	}
}
