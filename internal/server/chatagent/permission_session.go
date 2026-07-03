package chatagent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const sessionGrantsKeyPrefix = "session_grants:"

type permissionSessionEntry struct {
	state  *permission.SessionState
	loaded bool
	mu     sync.Mutex
}

// PermissionSessionManager keeps permission session state across runs for one chat session.
type PermissionSessionManager struct {
	states sync.Map
}

var permissionSessions PermissionSessionManager

// GetPermissionSession returns or creates session-scoped permission state.
func (m *PermissionSessionManager) GetPermissionSession(ctx context.Context, sessionID string) *permission.SessionState {
	if sessionID == "" {
		return permission.NewSessionState()
	}
	entry := m.getOrCreateEntry(sessionID)
	ensureSessionGrantsLoaded(ctx, sessionID, entry)
	return entry.state
}

func (m *PermissionSessionManager) getOrCreateEntry(sessionID string) *permissionSessionEntry {
	if raw, ok := m.states.Load(sessionID); ok {
		if entry, ok := raw.(*permissionSessionEntry); ok {
			return entry
		}
	}
	entry := &permissionSessionEntry{state: permission.NewSessionState()}
	actual, _ := m.states.LoadOrStore(sessionID, entry)
	if existing, ok := actual.(*permissionSessionEntry); ok {
		return existing
	}
	return entry
}

func ensureSessionGrantsLoaded(ctx context.Context, sessionID string, entry *permissionSessionEntry) {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.loaded {
		return
	}
	grants, err := loadSessionGrants(ctx, sessionID)
	if err != nil {
		flog.Debug("[chat-agent] load session grants session=%s: %v", sessionID, err)
	} else if len(grants) > 0 {
		entry.state.RestoreGrants(grants)
	}
	entry.loaded = true
}

// PersistSessionGrants writes the current session grants to storage.
func PersistSessionGrants(ctx context.Context, sessionID string, state *permission.SessionState) {
	if state == nil || sessionID == "" {
		return
	}
	if err := saveSessionGrants(ctx, sessionID, state.Grants()); err != nil {
		flog.Warn("[chat-agent] persist session grants session=%s: %v", sessionID, err)
	}
}

// ClearPermissionSession removes session permission state and persisted grants.
func (m *PermissionSessionManager) ClearPermissionSession(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	m.states.Delete(sessionID)
	if err := deleteSessionGrants(ctx, sessionID); err != nil {
		flog.Debug("[chat-agent] delete session grants session=%s: %v", sessionID, err)
	}
}

// ResetPermissionSessionsForTest clears all in-memory permission session state.
func ResetPermissionSessionsForTest() {
	permissionSessions = PermissionSessionManager{}
}

func sessionGrantsConfigKey(sessionID string) string {
	return sessionGrantsKeyPrefix + sessionID
}

func loadSessionGrants(ctx context.Context, sessionID string) (map[string][]string, error) {
	uid, err := SessionOwnerUID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if store.Database == nil {
		return nil, types.ErrUnavailable
	}
	raw, err := store.Database.ConfigGet(ctx, uid, PermissionTopic, sessionGrantsConfigKey(sessionID))
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load session grants: %w", err)
	}
	data, err := sonic.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal session grants: %w", err)
	}
	var grants map[string][]string
	if err := sonic.Unmarshal(data, &grants); err != nil {
		return nil, fmt.Errorf("parse session grants: %w", err)
	}
	return grants, nil
}

func saveSessionGrants(ctx context.Context, sessionID string, grants map[string][]string) error {
	uid, err := SessionOwnerUID(ctx, sessionID)
	if err != nil {
		return err
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if len(grants) == 0 {
		return deleteSessionGrants(ctx, sessionID)
	}
	data, err := sonic.Marshal(grants)
	if err != nil {
		return fmt.Errorf("marshal session grants: %w", err)
	}
	var payload map[string]any
	if err := sonic.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("session grants payload: %w", err)
	}
	return store.Database.ConfigSet(ctx, uid, PermissionTopic, sessionGrantsConfigKey(sessionID), types.KV(payload))
}

func deleteSessionGrants(ctx context.Context, sessionID string) error {
	uid, err := SessionOwnerUID(ctx, sessionID)
	if err != nil {
		return err
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	err = store.Database.ConfigDelete(ctx, uid, PermissionTopic, sessionGrantsConfigKey(sessionID))
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}
	return nil
}
