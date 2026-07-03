package chatagent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// PermissionTopic is the ConfigData topic for chat agent permissions.
	PermissionTopic = "chatagent"
	// PermissionKey is the ConfigData key for chat agent permissions.
	PermissionKey = "permission"
)

const permissionCacheTTL = 5 * time.Second

type permissionCacheEntry struct {
	config    permission.Config
	expiresAt time.Time
}

var (
	permissionCache   sync.Map
	permissionCacheMu sync.Mutex
)

// LoadUserPermissions reads merged effective permission config for one user.
func LoadUserPermissions(ctx context.Context, uid types.Uid) (permission.Config, error) {
	user, err := loadUserPermissionConfig(ctx, uid)
	if err != nil {
		return nil, err
	}
	return permission.EffectiveConfig(user), nil
}

func loadUserPermissionConfig(ctx context.Context, uid types.Uid) (permission.Config, error) {
	if uid.IsZero() {
		return permission.Config{}, nil
	}
	if cached, ok := loadPermissionCache(uid); ok {
		return cached, nil
	}
	if store.Database == nil {
		return permission.Config{}, types.ErrUnavailable
	}
	raw, err := store.Database.ConfigGet(ctx, uid, PermissionTopic, PermissionKey)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			storePermissionCache(uid, permission.Config{})
			return permission.Config{}, nil
		}
		return nil, fmt.Errorf("load user permissions: %w", err)
	}
	data, err := sonic.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal user permissions: %w", err)
	}
	cfg, err := permission.ParseConfig(data)
	if err != nil {
		return nil, err
	}
	storePermissionCache(uid, cfg)
	return cfg, nil
}

// SaveUserPermissions persists one user's permission overrides.
func SaveUserPermissions(ctx context.Context, uid types.Uid, cfg permission.Config) error {
	if err := permission.ValidateUserConfig(cfg); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	data, err := sonic.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}
	var payload map[string]any
	if err := sonic.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("permissions payload: %w", err)
	}
	kv := types.KV(payload)
	if err := store.Database.ConfigSet(ctx, uid, PermissionTopic, PermissionKey, kv); err != nil {
		return err
	}
	invalidatePermissionCache(uid)
	return nil
}

// DeleteUserPermissions removes user permission overrides.
func DeleteUserPermissions(ctx context.Context, uid types.Uid) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if err := store.Database.ConfigDelete(ctx, uid, PermissionTopic, PermissionKey); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			invalidatePermissionCache(uid)
			return nil
		}
		return err
	}
	invalidatePermissionCache(uid)
	return nil
}

func loadPermissionCache(uid types.Uid) (permission.Config, bool) {
	raw, ok := permissionCache.Load(uid.String())
	if !ok {
		return nil, false
	}
	entry, ok := raw.(permissionCacheEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		permissionCache.Delete(uid.String())
		return nil, false
	}
	return entry.config, true
}

func storePermissionCache(uid types.Uid, cfg permission.Config) {
	permissionCache.Store(uid.String(), permissionCacheEntry{
		config:    cfg,
		expiresAt: time.Now().Add(permissionCacheTTL),
	})
}

func invalidatePermissionCache(uid types.Uid) {
	permissionCache.Delete(uid.String())
}

// ResetPermissionCacheForTest clears the in-memory permission cache.
func ResetPermissionCacheForTest() {
	permissionCache = sync.Map{}
}

// SessionOwnerUID resolves the owning user for one chat session.
func SessionOwnerUID(ctx context.Context, sessionID string) (types.Uid, error) {
	if store.Database == nil {
		return types.Uid(""), types.ErrUnavailable
	}
	row, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		return types.Uid(""), err
	}
	return types.Uid(row.UID), nil
}
