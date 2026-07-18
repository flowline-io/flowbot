package chatagent

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// DBStorage persists agent session trees in PostgreSQL.
type DBStorage struct {
	sessionID   string
	uid         types.Uid
	usageSource string
}

// NewDBStorage creates a session storage adapter for the given session flag, owner uid, and usage source.
func NewDBStorage(sessionID string, uid types.Uid, usageSource string) *DBStorage {
	return &DBStorage{
		sessionID:   sessionID,
		uid:         uid,
		usageSource: types.NormalizeTokenUsageSource(usageSource),
	}
}

// Append stores a tree entry and updates the session leaf pointer.
func (s *DBStorage) Append(ctx context.Context, entry session.TreeEntry) error {
	payload, err := session.MarshalEntry(entry)
	if err != nil {
		return fmt.Errorf("chatagent storage: marshal entry: %w", err)
	}
	var payloadMap map[string]any
	if err := sonic.Unmarshal(payload, &payloadMap); err != nil {
		return fmt.Errorf("chatagent storage: payload map: %w", err)
	}

	row := &gen.ChatSessionEntry{
		Flag:      entry.ID,
		SessionID: s.sessionID,
		ParentID:  entry.ParentID,
		EntryType: string(entry.Type),
		Payload:   payloadMap,
	}
	if err := store.Database.AppendChatSessionEntry(ctx, row); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] append entry session=%s entry=%s type=%s: %w",
			s.sessionID, entry.ID, entry.Type, err))
		return err
	}
	s.recordLLMUsage(ctx, entry)
	flog.Debug("[chat-agent] appended entry session=%s entry=%s type=%s parent=%s",
		s.sessionID, entry.ID, entry.Type, entry.ParentID)
	return nil
}

func (s *DBStorage) recordLLMUsage(ctx context.Context, entry session.TreeEntry) {
	if s.uid.IsZero() || entry.Message == nil {
		return
	}
	assistant, ok := entry.Message.(msg.AssistantMessage)
	if !ok || assistant.Usage == nil {
		return
	}
	RecordLLMUsageMessages(ctx, s.uid, s.sessionID, s.usageSource, []agent.AgentMessage{assistant})
}

// GetBranch returns the ordered path from root to the requested leaf.
// Entries are loaded once via ListEntries, then walked in memory to avoid
// one DB round-trip per node on the active branch.
func (s *DBStorage) GetBranch(ctx context.Context, leafID string) ([]session.TreeEntry, error) {
	if leafID == "" {
		var err error
		leafID, err = s.GetLeafID(ctx)
		if err != nil {
			return nil, err
		}
		if leafID == "" {
			return nil, nil
		}
	}
	entries, err := s.ListEntries(ctx)
	if err != nil {
		return nil, err
	}
	return walkBranchFromEntries(entries, leafID)
}

// GetLeafID returns the current leaf pointer for the session.
func (s *DBStorage) GetLeafID(ctx context.Context) (string, error) {
	sess, err := store.Database.GetChatSession(ctx, s.sessionID)
	if err != nil {
		return "", err
	}
	return sess.LeafID, nil
}

// SetLeafID updates the current leaf pointer for the session.
func (s *DBStorage) SetLeafID(ctx context.Context, id string) error {
	return store.Database.UpdateChatSessionLeaf(ctx, s.sessionID, id)
}

// ListEntries returns all entries for the session in storage order.
func (s *DBStorage) ListEntries(ctx context.Context) ([]session.TreeEntry, error) {
	rows, err := store.Database.ListChatSessionEntries(ctx, s.sessionID)
	if err != nil {
		return nil, err
	}
	entries := make([]session.TreeEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := rowToTreeEntry(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// walkBranchFromEntries returns the ordered path from root to leafID using an
// in-memory index of already-loaded session entries.
func walkBranchFromEntries(entries []session.TreeEntry, leafID string) ([]session.TreeEntry, error) {
	if leafID == "" {
		return nil, nil
	}

	byID := make(map[string]session.TreeEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}

	path := make([]session.TreeEntry, 0, 16)
	currentID := leafID
	for currentID != "" {
		entry, ok := byID[currentID]
		if !ok {
			return nil, fmt.Errorf("chatagent storage: load entry %q: %w", currentID, types.ErrNotFound)
		}
		path = append([]session.TreeEntry{entry}, path...)
		currentID = entry.ParentID
	}
	return path, nil
}

func rowToTreeEntry(row *gen.ChatSessionEntry) (session.TreeEntry, error) {
	payload, err := sonic.Marshal(row.Payload)
	if err != nil {
		return session.TreeEntry{}, fmt.Errorf("chatagent storage: marshal payload: %w", err)
	}
	entry, err := session.UnmarshalEntry(payload)
	if err != nil {
		return session.TreeEntry{}, err
	}
	// Prefer DB columns for tree links; payload keys may use alternate spellings.
	if row.Flag != "" {
		entry.ID = row.Flag
	}
	entry.ParentID = row.ParentID
	return entry, nil
}
