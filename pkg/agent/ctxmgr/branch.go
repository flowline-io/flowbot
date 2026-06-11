package ctxmgr

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/tmc/langchaingo/llms"
)

// BranchSummaryResult is the outcome of branch summarization during tree navigation.
type BranchSummaryResult struct {
	Summary       string
	ReadFiles     []string
	ModifiedFiles []string
}

type branchEntriesResult struct {
	Entries        []session.TreeEntry
	CommonAncestor string
}

// CollectBranchEntries returns entries abandoned when moving from oldLeaf to newEntryID.
func CollectBranchEntries(allEntries []session.TreeEntry, oldLeafID, newEntryID string) result.Result[branchEntriesResult, result.BranchSummaryError] {
	if oldLeafID == "" || newEntryID == "" {
		return result.Err[branchEntriesResult, result.BranchSummaryError](
			result.NewBranchSummaryError("invalid_session", "empty branch navigation id", nil),
		)
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
	return result.Ok[branchEntriesResult, result.BranchSummaryError](branchEntriesResult{
		Entries:        abandoned,
		CommonAncestor: common,
	})
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
) result.Result[*BranchSummaryResult, result.BranchSummaryError] {
	if len(messages) == 0 {
		return result.Ok[*BranchSummaryResult, result.BranchSummaryError](&BranchSummaryResult{})
	}
	summaryResult := generateBranchSummary(ctx, model, modelName, messages, settings)
	if !summaryResult.IsOk() {
		return result.Err[*BranchSummaryResult, result.BranchSummaryError](
			result.NewBranchSummaryError(summaryResult.ErrorValue().Code(), summaryResult.ErrorValue().Message, summaryResult.ErrorValue().Cause),
		)
	}
	readFiles, modifiedFiles := ComputeFileLists(fileOps)
	summary := summaryResult.Value() + FormatFileOperations(readFiles, modifiedFiles)
	return result.Ok[*BranchSummaryResult, result.BranchSummaryError](&BranchSummaryResult{
		Summary:       normalizeSummary(summary),
		ReadFiles:     readFiles,
		ModifiedFiles: modifiedFiles,
	})
}

func generateBranchSummary(
	ctx context.Context,
	model llms.Model,
	modelName string,
	messages []msg.AgentMessage,
	settings Settings,
) result.Result[string, result.BranchSummaryError] {
	compactionResult := generateSummary(ctx, model, modelName, messages, "", summarizationPrompt, settings)
	if !compactionResult.IsOk() {
		compErr := compactionResult.ErrorValue()
		return result.Err[string, result.BranchSummaryError](
			result.NewBranchSummaryError(compErr.Code(), compErr.Message, compErr.Cause),
		)
	}
	return result.Ok[string, result.BranchSummaryError](compactionResult.Value())
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
