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

// Session manages a branchable conversation tree backed by storage.
type Session struct {
	storage Storage
}

// New creates a session backed by the given storage implementation.
func New(storage Storage) *Session {
	return &Session{storage: storage}
}

// Append stores a new tree entry and moves the leaf pointer to it.
func (s *Session) Append(ctx context.Context, entry TreeEntry) error {
	if err := s.storage.Append(ctx, entry); err != nil {
		return fmt.Errorf("session: append: %w", err)
	}
	return s.storage.SetLeafID(ctx, entry.ID)
}

// GetBranch returns the path from root to the given leaf, inclusive.
func (s *Session) GetBranch(ctx context.Context, leafID string) ([]TreeEntry, error) {
	if leafID == "" {
		var err error
		leafID, err = s.storage.GetLeafID(ctx)
		if err != nil {
			return nil, fmt.Errorf("session: get leaf: %w", err)
		}
	}
	branch, err := s.storage.GetBranch(ctx, leafID)
	if err != nil {
		return nil, fmt.Errorf("session: get branch: %w", err)
	}
	return orderBranch(branch), nil
}

// BuildContext reconstructs loop context from a branch path.
func BuildContext(path []TreeEntry) Context {
	ctx := Context{}
	for _, entry := range path {
		switch entry.Type {
		case EntryModelChange:
			ctx.ModelName = entry.ModelName
		case EntryActiveToolsChange:
			ctx.ActiveTools = append([]string(nil), entry.ActiveToolNames...)
		case EntryMessage:
			if entry.Message != nil {
				ctx.Messages = append(ctx.Messages, entry.Message)
			}
		case EntryBranchSummary:
			ctx.Messages = append(ctx.Messages, msg.BranchSummaryMessage{
				Summary: entry.Summary,
				FromID:  entry.FromID,
			})
		case EntryCompaction:
			ctx.Messages = append(ctx.Messages, msg.CompactionSummaryMessage{
				Summary:      entry.Summary,
				TokensBefore: entry.TokensBefore,
			})
		}
	}
	return ctx
}

// MoveTo switches the active leaf and optionally appends a branch summary node.
func (s *Session) MoveTo(ctx context.Context, entryID, summary string) error {
	if entryID == "" {
		return fmt.Errorf("session: empty entry id")
	}
	if summary != "" {
		summaryEntry := TreeEntry{
			ID:       entryID + "-summary",
			ParentID: entryID,
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
