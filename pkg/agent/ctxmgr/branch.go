package ctxmgr

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/tmc/langchaingo/llms"
)

// BranchSummaryResult is the outcome of branch summarization during tree navigation.
type BranchSummaryResult struct {
	Summary       string
	ReadFiles     []string
	ModifiedFiles []string
}

// CollectBranchEntries returns entries abandoned when moving from oldLeaf to newEntryID.
func CollectBranchEntries(allEntries []session.TreeEntry, oldLeafID, newEntryID string) ([]session.TreeEntry, string, error) {
	if oldLeafID == "" || newEntryID == "" {
		return nil, "", fmt.Errorf("ctxmgr: empty branch navigation id")
	}
	byID := make(map[string]session.TreeEntry, len(allEntries))
	for _, entry := range allEntries {
		byID[entry.ID] = entry
	}

	oldPath := pathToRoot(byID, oldLeafID)
	newPath := pathToRoot(byID, newEntryID)
	common := commonAncestor(oldPath, newPath)

	abandoned := make([]session.TreeEntry, 0)
	for _, entry := range oldPath {
		if entry.ID == common {
			break
		}
		abandoned = append(abandoned, entry)
	}
	return abandoned, common, nil
}

// PrepareBranchSummary selects messages to summarize within a token budget.
func PrepareBranchSummary(entries []session.TreeEntry, contextWindow int, settings Settings) ([]msg.AgentMessage, FileOperations, int) {
	settings = settings.WithDefaults()
	budget := contextWindow - settings.ReserveTokens
	if budget <= 0 {
		budget = settings.KeepRecentTokens
	}

	selected := make([]session.TreeEntry, 0, len(entries))
	totalTokens := 0
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		message, ok := messageFromEntry(entry)
		if !ok {
			continue
		}
		tokens := EstimateTokens(message)
		if totalTokens+tokens > budget && len(selected) > 0 {
			break
		}
		selected = append([]session.TreeEntry{entry}, selected...)
		totalTokens += tokens
	}

	messages := make([]msg.AgentMessage, 0, len(selected))
	for _, entry := range selected {
		message, ok := messageFromEntry(entry)
		if ok {
			messages = append(messages, message)
		}
	}
	fileOps := NewFileOperations()
	for _, message := range messages {
		ExtractFileOpsFromMessage(message, fileOps)
	}
	return messages, fileOps, totalTokens
}

// RunBranchSummary generates a summary for abandoned branch entries.
func RunBranchSummary(
	ctx context.Context,
	model llms.Model,
	modelName string,
	messages []msg.AgentMessage,
	fileOps FileOperations,
	settings Settings,
) (*BranchSummaryResult, error) {
	if len(messages) == 0 {
		return &BranchSummaryResult{}, nil
	}
	summary, err := generateSummary(ctx, model, modelName, messages, "", summarizationPrompt, settings)
	if err != nil {
		return nil, err
	}
	readFiles, modifiedFiles := ComputeFileLists(fileOps)
	summary += FormatFileOperations(readFiles, modifiedFiles)
	return &BranchSummaryResult{
		Summary:       normalizeSummary(summary),
		ReadFiles:     readFiles,
		ModifiedFiles: modifiedFiles,
	}, nil
}

func pathToRoot(byID map[string]session.TreeEntry, leafID string) []session.TreeEntry {
	path := make([]session.TreeEntry, 0)
	current, ok := byID[leafID]
	if !ok {
		return path
	}
	for {
		path = append([]session.TreeEntry{current}, path...)
		if current.ParentID == "" {
			break
		}
		next, ok := byID[current.ParentID]
		if !ok {
			break
		}
		current = next
	}
	return path
}

func commonAncestor(oldPath, newPath []session.TreeEntry) string {
	limit := min(len(newPath), len(oldPath))
	for i := 0; i < limit; i++ {
		if oldPath[i].ID != newPath[i].ID {
			if i == 0 {
				return ""
			}
			return oldPath[i-1].ID
		}
	}
	if limit == 0 {
		return ""
	}
	return oldPath[limit-1].ID
}
