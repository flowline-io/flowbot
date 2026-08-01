package session

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const (
	branchSummaryPrefix = "The following is a summary of a branch that this conversation came back from:\n\n<summary>\n"
	branchSummarySuffix = "</summary>"
)

// CompactionResult carries data needed to append a compaction tree node.
type CompactionResult struct {
	EntryID          string
	Summary          string
	FirstKeptEntryID string
	TokensBefore     int
	ReadFiles        []string
	ModifiedFiles    []string
}

// Session manages a branchable conversation tree backed by storage.
type Session struct {
	storage Storage
	cache   *branchCache
}

type branchCache struct {
	leafID string
	branch []TreeEntry
}

// New creates a session backed by the given storage implementation.
func New(storage Storage) *Session {
	return &Session{storage: storage}
}

func (s *Session) invalidateBranchCache() {
	s.cache = nil
}

func cloneBranch(branch []TreeEntry) []TreeEntry {
	if len(branch) == 0 {
		return nil
	}
	return append([]TreeEntry(nil), branch...)
}

// LeafID returns the current leaf pointer without loading the full branch.
func (s *Session) LeafID(ctx context.Context) (string, error) {
	leafID, err := s.storage.GetLeafID(ctx)
	if err != nil {
		return "", fmt.Errorf("session: get leaf id: %w", err)
	}
	return leafID, nil
}

// Append stores a new tree entry and moves the leaf pointer to it.
func (s *Session) Append(ctx context.Context, entry TreeEntry) error {
	s.invalidateBranchCache()
	if err := s.storage.Append(ctx, entry); err != nil {
		return fmt.Errorf("session: append: %w", err)
	}
	return s.storage.SetLeafID(ctx, entry.ID)
}

// AppendCompaction stores a compaction node and moves the leaf pointer to it.
func (s *Session) AppendCompaction(ctx context.Context, result CompactionResult) error {
	parentID, err := s.storage.GetLeafID(ctx)
	if err != nil {
		return fmt.Errorf("session: compaction parent: %w", err)
	}
	entryID := result.EntryID
	if entryID == "" {
		return fmt.Errorf("session: empty compaction entry id")
	}
	entry := TreeEntry{
		ID:               entryID,
		ParentID:         parentID,
		Type:             EntryCompaction,
		Summary:          result.Summary,
		FirstKeptEntryID: result.FirstKeptEntryID,
		TokensBefore:     result.TokensBefore,
		ReadFiles:        append([]string(nil), result.ReadFiles...),
		ModifiedFiles:    append([]string(nil), result.ModifiedFiles...),
	}
	return s.Append(ctx, entry)
}

// ListEntries returns all persisted session entries when supported by storage.
func (s *Session) ListEntries(ctx context.Context) ([]TreeEntry, error) {
	if lister, ok := s.storage.(interface {
		ListEntries(context.Context) ([]TreeEntry, error)
	}); ok {
		entries, err := lister.ListEntries(ctx)
		if err != nil {
			return nil, fmt.Errorf("session: list entries: %w", err)
		}
		return entries, nil
	}
	return s.GetBranch(ctx, "")
}

// GetBranch returns the path from root to the given leaf, inclusive.
func (s *Session) GetBranch(ctx context.Context, leafID string) ([]TreeEntry, error) {
	effectiveLeaf := leafID
	if effectiveLeaf == "" {
		var err error
		effectiveLeaf, err = s.storage.GetLeafID(ctx)
		if err != nil {
			return nil, fmt.Errorf("session: get leaf: %w", err)
		}
		if effectiveLeaf == "" {
			s.invalidateBranchCache()
			return nil, nil
		}
	}
	if s.cache != nil && s.cache.leafID == effectiveLeaf {
		return cloneBranch(s.cache.branch), nil
	}

	branch, err := s.storage.GetBranch(ctx, effectiveLeaf)
	if err != nil {
		return nil, fmt.Errorf("session: get branch: %w", err)
	}
	ordered := orderBranch(branch)
	s.cache = &branchCache{
		leafID: effectiveLeaf,
		branch: cloneBranch(ordered),
	}
	return cloneBranch(ordered), nil
}

// BuildContext reconstructs loop context from a branch path with compaction boundaries.
func BuildContext(path []TreeEntry) Context {
	ctx := Context{}
	if len(path) == 0 {
		return ctx
	}

	compaction := applyPathMetadata(path, &ctx)
	if compaction != nil {
		appendCompactionContext(&ctx, path, compaction)
		ctx.Messages = SanitizeToolMessageOrder(ctx.Messages)
		return ctx
	}

	for _, entry := range path {
		appendContextMessage(&ctx, entry)
	}
	ctx.Messages = SanitizeToolMessageOrder(ctx.Messages)
	return ctx
}

// SanitizeToolMessageOrder rebuilds LLM-valid assistant/tool pairing.
// Mid-turn persistence may store tool results before their assistant tool_calls;
// reload paths must not send that order to the provider.
func SanitizeToolMessageOrder(messages []msg.AgentMessage) []msg.AgentMessage {
	if len(messages) < 2 {
		return messages
	}

	toolsByID := make(map[string]msg.ToolResultMessage)
	for _, message := range messages {
		tr, ok := message.(msg.ToolResultMessage)
		if !ok || tr.ToolCallID == "" {
			continue
		}
		toolsByID[tr.ToolCallID] = tr
	}

	out := make([]msg.AgentMessage, 0, len(messages))
	emittedTools := make(map[string]struct{})

	for _, message := range messages {
		switch m := message.(type) {
		case msg.ToolResultMessage:
			// Re-emitted immediately after the owning assistant.
			continue
		case msg.AssistantMessage:
			calls := m.ToolCalls()
			if len(calls) == 0 {
				out = append(out, m)
				continue
			}
			if assistantToolsAlreadyEmitted(calls, emittedTools) {
				continue
			}
			out = append(out, m)
			for _, call := range calls {
				if call.ID == "" {
					continue
				}
				if tr, ok := toolsByID[call.ID]; ok {
					out = append(out, tr)
				} else {
					out = append(out, msg.ToolResultMessage{
						ToolCallID: call.ID,
						Name:       call.Name,
						Parts:      []msg.ContentPart{msg.TextPart{Text: "tool result unavailable"}},
						IsError:    true,
					})
				}
				emittedTools[call.ID] = struct{}{}
			}
		default:
			out = append(out, m)
		}
	}
	return out
}

func assistantToolsAlreadyEmitted(calls []msg.ToolCallPart, emitted map[string]struct{}) bool {
	if len(emitted) == 0 {
		return false
	}
	sawID := false
	for _, call := range calls {
		if call.ID == "" {
			continue
		}
		sawID = true
		if _, ok := emitted[call.ID]; !ok {
			return false
		}
	}
	return sawID
}

// MoveTo switches the active leaf and optionally appends a branch summary node.
func (s *Session) MoveTo(ctx context.Context, entryID, summary string) error {
	if entryID == "" {
		return fmt.Errorf("session: empty entry id")
	}
	s.invalidateBranchCache()
	if summary != "" {
		parentID := entryID
		summaryEntry := TreeEntry{
			ID:       entryID + "-summary",
			ParentID: parentID,
			Type:     EntryBranchSummary,
			Summary:  summary,
			FromID:   entryID,
		}
		if err := s.storage.Append(ctx, summaryEntry); err != nil {
			return err
		}
		return s.storage.SetLeafID(ctx, summaryEntry.ID)
	}
	return s.storage.SetLeafID(ctx, entryID)
}

// ToAgentContext converts a session context into an agent loop context.
func ToAgentContext(sessionCtx Context, systemPrompt string) *msg.Context {
	return &msg.Context{
		SystemPrompt: systemPrompt,
		Messages:     append([]msg.AgentMessage(nil), sessionCtx.Messages...),
		ModelName:    sessionCtx.ModelName,
	}
}

func indexOfEntry(entries []TreeEntry, id string) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

func orderBranch(entries []TreeEntry) []TreeEntry {
	if len(entries) <= 1 {
		return entries
	}
	byID := make(map[string]TreeEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	leaf := entries[len(entries)-1]
	path := []TreeEntry{leaf}
	current := leaf
	for current.ParentID != "" {
		parent, ok := byID[current.ParentID]
		if !ok {
			break
		}
		path = append([]TreeEntry{parent}, path...)
		current = parent
	}
	return path
}
