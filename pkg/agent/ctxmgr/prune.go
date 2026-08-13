package ctxmgr

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/google/uuid"
)

const (
	toolResultPruneThresholdRunes = 8192
	toolResultPruneHeadRunes      = 4096
	toolResultPruneTailRunes      = 1024
	toolResultPruneMarker         = "\n\n[... tool result middle pruned ...]\n\n"
)

// PruneToolOutputs rewrites oversized tool-result payloads to a bounded head,
// omission marker, and tail. Messages that are not over budget are unchanged.
func PruneToolOutputs(messages []msg.AgentMessage, settings Settings) []msg.AgentMessage {
	if !settings.PruneToolOutputs || len(messages) == 0 {
		return messages
	}

	out := make([]msg.AgentMessage, 0, len(messages))
	changed := false
	for _, message := range messages {
		toolResult, ok := message.(msg.ToolResultMessage)
		if !ok {
			out = append(out, message)
			continue
		}
		pruned, did := pruneToolResultMessage(toolResult)
		if !did {
			out = append(out, message)
			continue
		}
		changed = true
		out = append(out, pruned)
	}
	if !changed {
		return messages
	}
	return out
}

func pruneToolResultMessage(toolResult msg.ToolResultMessage) (msg.ToolResultMessage, bool) {
	text, nonText := splitToolResultParts(toolResult.Parts)
	prunedText, did := pruneToolResultText(text)
	if !did {
		return toolResult, false
	}
	parts := make([]msg.ContentPart, 0, 1+len(nonText))
	if prunedText != "" {
		parts = append(parts, msg.TextPart{Text: prunedText})
	}
	parts = append(parts, nonText...)
	next := toolResult
	next.Parts = parts
	return next, true
}

func pruneToolResultText(text string) (string, bool) {
	runes := []rune(text)
	if len(runes) <= toolResultPruneThresholdRunes {
		return text, false
	}
	if toolResultPruneHeadRunes+toolResultPruneTailRunes >= len(runes) {
		return text, false
	}
	marker := []rune(toolResultPruneMarker)
	head := runes[:toolResultPruneHeadRunes]
	tail := runes[len(runes)-toolResultPruneTailRunes:]
	out := make([]rune, 0, len(head)+len(marker)+len(tail))
	out = append(out, head...)
	out = append(out, marker...)
	out = append(out, tail...)
	if len(out) >= len(runes) {
		return text, false
	}
	return string(out), true
}

func splitToolResultParts(parts []msg.ContentPart) (text string, nonText []msg.ContentPart) {
	var b []rune
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			b = append(b, []rune(tp.Text)...)
			continue
		}
		nonText = append(nonText, part)
	}
	return string(b), nonText
}

func persistPrunedToolResults(
	ctx context.Context,
	sess *session.Session,
	path []session.TreeEntry,
	settings Settings,
) (bool, error) {
	if sess == nil || !settings.PruneToolOutputs {
		return false, nil
	}

	pruned := false
	for _, entry := range latestVisibleToolResultEntries(path) {
		toolResult, ok := entry.Message.(msg.ToolResultMessage)
		if !ok {
			continue
		}
		next, did := pruneToolResultMessage(toolResult)
		if !did {
			continue
		}
		leafID, err := sess.LeafID(ctx)
		if err != nil {
			return pruned, fmt.Errorf("ctxmgr: load prune leaf: %w", err)
		}
		if err := sess.Append(ctx, session.TreeEntry{
			ID:       "prune-" + uuid.NewString(),
			ParentID: leafID,
			Type:     session.EntryMessage,
			Message:  next,
		}); err != nil {
			return pruned, fmt.Errorf("ctxmgr: append pruned tool result: %w", err)
		}
		pruned = true
	}
	return pruned, nil
}

func latestVisibleToolResultEntries(path []session.TreeEntry) []session.TreeEntry {
	visible := visibleMessageEntries(path)
	seen := make(map[string]struct{}, len(visible))
	latest := make([]session.TreeEntry, 0, len(visible))
	for i := len(visible) - 1; i >= 0; i-- {
		entry := visible[i]
		toolResult, ok := entry.Message.(msg.ToolResultMessage)
		if !ok {
			continue
		}
		key := toolResult.ToolCallID
		if key == "" {
			key = entry.ID
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		latest = append(latest, entry)
	}
	return latest
}

func visibleMessageEntries(path []session.TreeEntry) []session.TreeEntry {
	start := 0
	compactionIdx := lastCompactionIndex(path)
	if compactionIdx >= 0 {
		start = firstKeptStartIndex(path, path[compactionIdx], compactionIdx+1)
	}
	out := make([]session.TreeEntry, 0, len(path)-start)
	for i := start; i < len(path); i++ {
		entry := path[i]
		if entry.Type != session.EntryMessage || entry.Message == nil {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func latestToolResultsByCallID(entries []session.TreeEntry) map[string]msg.ToolResultMessage {
	latest := make(map[string]msg.ToolResultMessage)
	for _, entry := range latestVisibleToolResultEntries(entries) {
		toolResult, ok := entry.Message.(msg.ToolResultMessage)
		if !ok || toolResult.ToolCallID == "" {
			continue
		}
		latest[toolResult.ToolCallID] = toolResult
	}
	return latest
}

func overlayToolResults(messages []msg.AgentMessage, latest map[string]msg.ToolResultMessage) []msg.AgentMessage {
	if len(latest) == 0 || len(messages) == 0 {
		return messages
	}
	out := append([]msg.AgentMessage(nil), messages...)
	for i, message := range out {
		toolResult, ok := message.(msg.ToolResultMessage)
		if !ok || toolResult.ToolCallID == "" {
			continue
		}
		if newer, ok := latest[toolResult.ToolCallID]; ok {
			out[i] = newer
		}
	}
	return out
}
