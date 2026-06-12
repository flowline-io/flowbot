package chatagent

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// DBStorage persists agent session trees in PostgreSQL.
type DBStorage struct {
	sessionID string
}

// NewDBStorage creates a session storage adapter for the given session flag.
func NewDBStorage(sessionID string) *DBStorage {
	return &DBStorage{sessionID: sessionID}
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
	flog.Debug("[chat-agent] appended entry session=%s entry=%s type=%s parent=%s",
		s.sessionID, entry.ID, entry.Type, entry.ParentID)
	return nil
}

// GetBranch returns the ordered path from root to the requested leaf.
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
	return walkBranchFromLeaf(ctx, s.sessionID, leafID, store.Database.GetChatSessionEntryInSession)
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

type sessionEntryGetter func(ctx context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error)

func walkBranchFromLeaf(
	ctx context.Context,
	sessionID, leafID string,
	getEntry sessionEntryGetter,
) ([]session.TreeEntry, error) {
	if leafID == "" {
		return nil, nil
	}

	path := make([]session.TreeEntry, 0, 16)
	currentID := leafID
	for currentID != "" {
		row, err := getEntry(ctx, sessionID, currentID)
		if err != nil {
			return nil, fmt.Errorf("chatagent storage: load entry %q: %w", currentID, err)
		}
		entry, err := rowToTreeEntry(row)
		if err != nil {
			return nil, err
		}
		path = append([]session.TreeEntry{entry}, path...)
		currentID = row.ParentID
	}
	return path, nil
}

func rowToTreeEntry(row *gen.ChatSessionEntry) (session.TreeEntry, error) {
	payload, err := sonic.Marshal(row.Payload)
	if err != nil {
		return session.TreeEntry{}, fmt.Errorf("chatagent storage: marshal payload: %w", err)
	}
	return session.UnmarshalEntry(payload)
}
