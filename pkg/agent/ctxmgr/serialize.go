package ctxmgr

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

const toolResultMaxChars = 2000

const summarizationSystemPrompt = `You are a context summarization assistant. Your task is to read a conversation between a user and an AI assistant, then produce a structured summary following the exact format specified.

Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`

const summarizationPrompt = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

const updateSummarizationPrompt = `The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.

Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]

### In Progress
- [ ] [Current work - update based on progress]

### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

const turnPrefixSummarizationPrompt = `This is the PREFIX of a turn that was too large to keep. The SUFFIX (recent work) is retained.

Summarize the prefix to provide context for the retained suffix:

## Original Request
[What did the user ask for in this turn?]

## Early Progress
- [Key decisions and work done in the prefix]

## Context for Suffix
- [Information needed to understand the retained recent work]

Be concise. Focus on what's needed to understand the kept suffix.`

// SerializeConversation converts agent messages to plain text for summarization prompts.
func SerializeConversation(messages []msg.AgentMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		if serialized := serializeOneMessage(message); serialized != "" {
			parts = append(parts, serialized)
		}
	}
	return strings.Join(parts, "\n\n")
}

func buildSummarizationPrompt(messages []msg.AgentMessage, previousSummary string, basePrompt string) (string, error) {
	conversationText := SerializeConversation(messages)
	var b strings.Builder
	_, _ = b.WriteString("<conversation>\n")
	_, _ = b.WriteString(conversationText)
	_, _ = b.WriteString("\n</conversation>\n\n")
	if previousSummary != "" {
		_, _ = b.WriteString("<previous-summary>\n")
		_, _ = b.WriteString(previousSummary)
		_, _ = b.WriteString("\n</previous-summary>\n\n")
	}
	_, _ = b.WriteString(basePrompt)
	return b.String(), nil
}

func messageFromEntry(entry session.TreeEntry) (msg.AgentMessage, bool) {
	switch entry.Type {
	case session.EntryMessage:
		if entry.Message != nil {
			return entry.Message, true
		}
	case session.EntryBranchSummary:
		return msg.BranchSummaryMessage{Summary: entry.Summary, FromID: entry.FromID}, true
	}
	return nil, false
}

func messageFromEntryForCompaction(entry session.TreeEntry) (msg.AgentMessage, bool) {
	if entry.Type == session.EntryCompaction {
		return nil, false
	}
	return messageFromEntry(entry)
}
