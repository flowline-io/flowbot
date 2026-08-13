package ctxmgr

import (
	"context"
	"errors"
	"strings"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// CompactionResult is the outcome of a successful compaction run.
type CompactionResult struct {
	Summary          string
	FirstKeptEntryID string
	TokensBefore     int
	ReadFiles        []string
	ModifiedFiles    []string
}

// CompactReport records whether a compaction pass changed the model-visible surface.
type CompactReport struct {
	Pruned     bool
	Summarized bool
}

// Changed reports whether prune or summarization changed the model-visible surface.
func (r CompactReport) Changed() bool {
	return r.Pruned || r.Summarized
}

// CompactionPreparation holds precomputed compaction inputs.
type CompactionPreparation struct {
	FirstKeptEntryID    string
	MessagesToSummarize []msg.AgentMessage
	TurnPrefixMessages  []msg.AgentMessage
	IsSplitTurn         bool
	TokensBefore        int
	PreviousSummary     string
	FileOps             FileOperations
	Settings            Settings
	SystemPrompt        string
	Tools               []llms.Tool
	ThinkingLevel       string
}

// ShouldCompact reports whether context usage exceeds the compaction threshold.
func ShouldCompact(contextTokens, contextWindow int, settings Settings) bool {
	settings = settings.WithDefaults()
	if !settings.Enabled || contextWindow <= 0 {
		return false
	}
	return contextTokens > contextWindow-settings.ReserveTokens
}

// PrepareCompaction computes which session entries will be summarized.
func PrepareCompaction(pathEntries []session.TreeEntry, settings Settings, opts PrepareOptions) result.Result[*CompactionPreparation, result.CompactionError] {
	settings = settings.WithDefaults()
	leafIsCompaction := len(pathEntries) > 0 && pathEntries[len(pathEntries)-1].Type == session.EntryCompaction
	if leafIsCompaction && !opts.Force {
		return result.Ok[*CompactionPreparation, result.CompactionError](nil)
	}

	bounds := computeCompactionBounds(pathEntries, settings, leafIsCompaction, opts.Force)
	messages := session.BuildContext(pathEntries).Messages
	tokensBefore := EstimateContextTokens(append(messages, opts.ExtraMessages...)).Tokens

	if bounds.boundaryEnd <= bounds.boundaryStart && len(opts.ExtraMessages) == 0 {
		return result.Ok[*CompactionPreparation, result.CompactionError](nil)
	}
	if bounds.boundaryEnd <= bounds.boundaryStart && len(opts.ExtraMessages) > 0 && leafIsCompaction {
		return result.Ok[*CompactionPreparation, result.CompactionError](prepareExtraOnlyCompaction(
			pathEntries, opts, bounds.prevCompactionIndex, bounds.previousSummary, tokensBefore, settings,
		))
	}

	return buildCompactionPreparation(pathEntries, bounds, opts, tokensBefore, settings)
}

func collectMessages(entries []session.TreeEntry, start, end int) []msg.AgentMessage {
	if start < 0 {
		start = 0
	}
	if end > len(entries) {
		end = len(entries)
	}
	messages := make([]msg.AgentMessage, 0, end-start)
	for i := start; i < end; i++ {
		message, ok := messageFromEntryForCompaction(entries[i])
		if ok {
			messages = append(messages, message)
		}
	}
	return messages
}

// RunCompaction generates a summary and returns compaction metadata.
func RunCompaction(
	ctx context.Context,
	model llms.Model,
	modelName string,
	preparation *CompactionPreparation,
) result.Result[*CompactionResult, result.CompactionError] {
	if preparation == nil {
		return result.Err[*CompactionResult, result.CompactionError](
			result.NewCompactionError("invalid_session", "nil compaction preparation", nil),
		)
	}

	req := summaryRequest{
		Model:         model,
		ModelName:     modelName,
		Settings:      preparation.Settings,
		SystemPrompt:  preparation.SystemPrompt,
		Tools:         preparation.Tools,
		ThinkingLevel: preparation.ThinkingLevel,
	}

	var summary string
	if preparation.IsSplitTurn && len(preparation.TurnPrefixMessages) > 0 {
		historySummary := "No prior history."
		if len(preparation.MessagesToSummarize) > 0 {
			req.Messages = preparation.MessagesToSummarize
			req.PreviousSummary = preparation.PreviousSummary
			req.BasePrompt = summarizationPrompt
			historyResult := generateSummary(ctx, req)
			if !historyResult.IsOk() {
				return result.Err[*CompactionResult, result.CompactionError](historyResult.ErrorValue())
			}
			historySummary = historyResult.Value()
		}
		req.Messages = preparation.TurnPrefixMessages
		req.PreviousSummary = ""
		req.BasePrompt = turnPrefixSummarizationPrompt
		turnPrefixResult := generateSummary(ctx, req)
		if !turnPrefixResult.IsOk() {
			return result.Err[*CompactionResult, result.CompactionError](turnPrefixResult.ErrorValue())
		}
		summary = historySummary + "\n\n## Turn Prefix Summary\n" + turnPrefixResult.Value()
	} else if len(preparation.MessagesToSummarize) > 0 {
		req.Messages = preparation.MessagesToSummarize
		req.PreviousSummary = preparation.PreviousSummary
		req.BasePrompt = summarizationPrompt
		if preparation.PreviousSummary != "" {
			req.BasePrompt = updateSummarizationPrompt
		}
		summaryResult := generateSummary(ctx, req)
		if !summaryResult.IsOk() {
			return result.Err[*CompactionResult, result.CompactionError](summaryResult.ErrorValue())
		}
		summary = summaryResult.Value()
	} else {
		return result.Err[*CompactionResult, result.CompactionError](
			result.NewCompactionError("nothing_to_compact", "nothing to summarize", nil),
		)
	}

	readFiles, modifiedFiles := ComputeFileLists(preparation.FileOps)
	summary += FormatFileOperations(readFiles, modifiedFiles)

	return result.Ok[*CompactionResult, result.CompactionError](&CompactionResult{
		Summary:          normalizeSummary(summary),
		FirstKeptEntryID: preparation.FirstKeptEntryID,
		TokensBefore:     preparation.TokensBefore,
		ReadFiles:        readFiles,
		ModifiedFiles:    modifiedFiles,
	})
}

type summaryRequest struct {
	Model           llms.Model
	ModelName       string
	Messages        []msg.AgentMessage
	PreviousSummary string
	BasePrompt      string
	Settings        Settings
	SystemPrompt    string
	Tools           []llms.Tool
	ThinkingLevel   string
}

func generateSummary(ctx context.Context, req summaryRequest) result.Result[string, result.CompactionError] {
	req.Settings = req.Settings.WithDefaults()
	replay := req.Messages
	if req.PreviousSummary != "" {
		replay = append([]msg.AgentMessage{msg.CompactionSummaryMessage{Summary: req.PreviousSummary}}, replay...)
	}
	replay = session.SanitizeToolMessageOrder(replay)
	llmMessages, err := transform.DefaultConvertToLLM(replay)
	if err != nil {
		return result.Err[string, result.CompactionError](
			result.NewCompactionError("invalid_session", "build summarization prompt", err),
		)
	}
	llmMessages = append(llmMessages, llms.TextParts(llms.ChatMessageTypeHuman, compactionInstruction(req.BasePrompt)))
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return result.Err[string, result.CompactionError](
			result.NewCompactionError("aborted", "summarization aborted", ctx.Err()),
		)
	}
	maxTokens := req.Settings.ReserveTokens * 4 / 5
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	content, err := agentllm.CompleteWithTools(ctx, agentllm.CompleteRequest{
		Model:                  req.Model,
		SystemPrompt:           req.SystemPrompt,
		Messages:               llmMessages,
		ModelName:              req.ModelName,
		MaxTokens:              maxTokens,
		Tools:                  req.Tools,
		ThinkingLevel:          req.ThinkingLevel,
		AssistantToolReasoning: collectAssistantToolReasoning(replay),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, agentllm.ErrAborted) {
			return result.Err[string, result.CompactionError](
				result.NewCompactionError("aborted", "summarization aborted", err),
			)
		}
		return result.Err[string, result.CompactionError](
			result.NewCompactionError("summarization_failed", "summarization failed", err),
		)
	}
	if normalizeSummary(content) == "" {
		return result.Err[string, result.CompactionError](
			result.NewCompactionError("summarization_failed", "empty summarization response", nil),
		)
	}
	return result.Ok[string, result.CompactionError](content)
}

func collectAssistantToolReasoning(messages []msg.AgentMessage) map[string]string {
	out := map[string]string{}
	for _, message := range messages {
		assistant, ok := message.(msg.AssistantMessage)
		if !ok {
			continue
		}
		for _, call := range assistant.ToolCalls() {
			id := strings.TrimSpace(call.ID)
			if id == "" {
				continue
			}
			out[id] = assistant.ThinkingText
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func indexOfEntry(entries []session.TreeEntry, id string) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

// NewCompactionEntryID returns a unique compaction node identifier.
func NewCompactionEntryID() string {
	return uuid.NewString()
}
