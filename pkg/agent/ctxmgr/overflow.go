package ctxmgr

import (
	"errors"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
)

var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)prompt is too long`),
	regexp.MustCompile(`(?i)request_too_large`),
	regexp.MustCompile(`(?i)exceeds the context window`),
	regexp.MustCompile(`(?i)exceeds (?:the )?(?:model'?s )?maximum context length of [\d,]+ tokens?`),
	regexp.MustCompile(`(?i)input token count.*exceeds the maximum`),
	regexp.MustCompile(`(?i)maximum prompt length is \d+`),
	regexp.MustCompile(`(?i)reduce the length of the messages`),
	regexp.MustCompile(`(?i)maximum context length is \d+ tokens`),
	regexp.MustCompile(`(?i)exceeds (?:the )?maximum allowed input length of [\d,]+ tokens?`),
	regexp.MustCompile(`(?i)exceeds the available context size`),
	regexp.MustCompile(`(?i)greater than the context length`),
	regexp.MustCompile(`(?i)context window exceeds limit`),
	regexp.MustCompile(`(?i)exceeded model token limit`),
	regexp.MustCompile(`(?i)too large for model with \d+ maximum context length`),
	regexp.MustCompile(`(?i)prompt too long; exceeded (?:max )?context length`),
	regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`),
	regexp.MustCompile(`(?i)too many tokens`),
	regexp.MustCompile(`(?i)token limit exceeded`),
}

var nonOverflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(Throttling error|Service unavailable):`),
	regexp.MustCompile(`(?i)rate limit`),
	regexp.MustCompile(`(?i)too many requests`),
}

// ErrSummarizationFailed indicates compaction summarization could not complete.
var ErrSummarizationFailed = result.NewCompactionError("summarization_failed", "summarization failed", nil)

// ErrCompactionRequired indicates context still exceeds budget but no history could be compacted.
var ErrCompactionRequired = result.NewCompactionError("nothing_to_compact", "compaction required but no removable history", nil)

// ErrBranchSummaryAborted indicates branch summarization was cancelled.
var ErrBranchSummaryAborted = result.NewBranchSummaryError("aborted", "branch summarization aborted", nil)

// IsContextOverflowErr reports whether an LLM error indicates context window overflow.
func IsContextOverflowErr(err error) bool {
	if err == nil {
		return false
	}
	var overflow result.OverflowError
	if errors.As(err, &overflow) {
		return true
	}
	errText := err.Error()
	if matchesAny(nonOverflowPatterns, errText) {
		return false
	}
	return matchesAny(overflowPatterns, errText)
}

// WrapOverflowError wraps an underlying error as a typed OverflowError when overflow is detected.
func WrapOverflowError(err error) error {
	if err == nil || !IsContextOverflowErr(err) {
		return err
	}
	return result.NewOverflowError("context window exceeded", err)
}

// IsContextOverflowMessage reports whether an assistant message indicates overflow.
func IsContextOverflowMessage(message msg.AssistantMessage, contextWindow int) bool {
	if message.StopReason == "error" {
		text := message.TextContent()
		if matchesAny(nonOverflowPatterns, text) {
			return false
		}
		if matchesAny(overflowPatterns, text) {
			return true
		}
	}
	if contextWindow <= 0 || message.Usage == nil {
		return false
	}
	inputTokens := message.Usage.PromptTokens + message.Usage.CacheRead
	if message.StopReason == "complete" && inputTokens > contextWindow {
		return true
	}
	if message.StopReason == "length" && message.Usage.CompletionTokens == 0 {
		if inputTokens >= int(float64(contextWindow)*0.99) {
			return true
		}
	}
	return false
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

// IsOverflowResult checks stream errors and the last assistant message for overflow.
func IsOverflowResult(err error, messages []msg.AgentMessage, contextWindow int) bool {
	if IsContextOverflowErr(err) {
		return true
	}
	for i := len(messages) - 1; i >= 0; i-- {
		assistant, ok := messages[i].(msg.AssistantMessage)
		if !ok {
			continue
		}
		return IsContextOverflowMessage(assistant, contextWindow)
	}
	return false
}

func normalizeSummary(text string) string {
	return strings.TrimSpace(text)
}
