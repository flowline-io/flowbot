package chatagent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// ApprovalKey is the ConfigData key for chat agent approval mode.
	ApprovalKey              = "approval"
	sessionApprovalKeyPrefix = "session_approval:"
)

const approvalCacheTTL = 5 * time.Second

type approvalCacheEntry struct {
	mode      approval.Mode
	set       bool
	expiresAt time.Time
}

var (
	approvalCache   sync.Map
	approvalCacheMu sync.Mutex
)

// LoadUserApprovalMode returns the effective approval mode: user DB → YAML → manual.
func LoadUserApprovalMode(ctx context.Context, uid types.Uid) (approval.Mode, error) {
	userMode, set, err := loadUserApprovalModeOverride(ctx, uid)
	if err != nil {
		return approval.ModeManual, err
	}
	if set {
		return userMode, nil
	}
	mode, err := approval.ParseMode(config.ChatAgentApprovalModeDefault())
	if err != nil {
		return approval.ModeManual, nil
	}
	return mode, nil
}

func loadUserApprovalModeOverride(ctx context.Context, uid types.Uid) (approval.Mode, bool, error) {
	if uid.IsZero() {
		return approval.ModeManual, false, nil
	}
	if cached, ok, hit := loadApprovalCache(uid); hit {
		return cached, ok, nil
	}
	if store.Database == nil {
		return approval.ModeManual, false, types.ErrUnavailable
	}
	raw, err := store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, PermissionTopic, ApprovalKey)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			storeApprovalCache(uid, approval.ModeManual, false)
			return approval.ModeManual, false, nil
		}
		return approval.ModeManual, false, fmt.Errorf("load user approval mode: %w", err)
	}
	modeRaw, ok := raw.String("mode")
	if !ok || modeRaw == "" {
		return approval.ModeManual, false, types.WrapError(types.ErrInvalidArgument, "approval mode missing", nil)
	}
	mode, err := approval.ParseMode(modeRaw)
	if err != nil {
		return approval.ModeManual, false, types.WrapError(types.ErrInvalidArgument, "invalid approval mode", err)
	}
	storeApprovalCache(uid, mode, true)
	return mode, true, nil
}

// SaveUserApprovalMode persists one user's approval mode override.
func SaveUserApprovalMode(ctx context.Context, uid types.Uid, mode approval.Mode) error {
	if !mode.Valid() {
		return types.Errorf(types.ErrInvalidArgument, "invalid approval mode %q", mode)
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	kv := types.KV{"mode": string(mode)}
	if err := store.ModuleDataStoreFromDB().ConfigSet(ctx, uid, PermissionTopic, ApprovalKey, kv); err != nil {
		return err
	}
	storeApprovalCache(uid, mode, true)
	return nil
}

// DeleteUserApprovalMode removes the user override so YAML/default applies.
func DeleteUserApprovalMode(ctx context.Context, uid types.Uid) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if err := store.ModuleDataStoreFromDB().ConfigDelete(ctx, uid, PermissionTopic, ApprovalKey); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			invalidateApprovalCache(uid)
			return nil
		}
		return err
	}
	invalidateApprovalCache(uid)
	return nil
}

func loadApprovalCache(uid types.Uid) (approval.Mode, bool, bool) {
	raw, ok := approvalCache.Load(uid.String())
	if !ok {
		return approval.ModeManual, false, false
	}
	entry, ok := raw.(approvalCacheEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		approvalCache.Delete(uid.String())
		return approval.ModeManual, false, false
	}
	return entry.mode, entry.set, true
}

func storeApprovalCache(uid types.Uid, mode approval.Mode, set bool) {
	approvalCache.Store(uid.String(), approvalCacheEntry{
		mode:      mode,
		set:       set,
		expiresAt: time.Now().Add(approvalCacheTTL),
	})
}

func invalidateApprovalCache(uid types.Uid) {
	approvalCache.Delete(uid.String())
}

// ResetApprovalCacheForTest clears the in-memory approval cache.
func ResetApprovalCacheForTest() {
	approvalCacheMu.Lock()
	defer approvalCacheMu.Unlock()
	approvalCache = sync.Map{}
}

func sessionApprovalConfigKey(sessionID string) string {
	return sessionApprovalKeyPrefix + sessionID
}

// LoadSessionApprovalMode returns a per-session approval override when set.
func LoadSessionApprovalMode(ctx context.Context, sessionID string) (approval.Mode, bool, error) {
	if sessionID == "" {
		return approval.ModeManual, false, nil
	}
	uid, err := SessionOwnerUID(ctx, sessionID)
	if err != nil {
		return approval.ModeManual, false, err
	}
	if store.Database == nil {
		return approval.ModeManual, false, types.ErrUnavailable
	}
	raw, err := store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, PermissionTopic, sessionApprovalConfigKey(sessionID))
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return approval.ModeManual, false, nil
		}
		return approval.ModeManual, false, fmt.Errorf("load session approval mode: %w", err)
	}
	modeRaw, ok := raw.String("mode")
	if !ok || modeRaw == "" {
		return approval.ModeManual, false, types.WrapError(types.ErrInvalidArgument, "session approval mode missing", nil)
	}
	mode, err := approval.ParseMode(modeRaw)
	if err != nil {
		return approval.ModeManual, false, types.WrapError(types.ErrInvalidArgument, "invalid session approval mode", err)
	}
	return mode, true, nil
}

// SaveSessionApprovalMode persists a per-session approval mode override.
func SaveSessionApprovalMode(ctx context.Context, sessionID string, mode approval.Mode) error {
	if !mode.Valid() {
		return types.Errorf(types.ErrInvalidArgument, "invalid approval mode %q", mode)
	}
	uid, err := SessionOwnerUID(ctx, sessionID)
	if err != nil {
		return err
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	return store.ModuleDataStoreFromDB().ConfigSet(ctx, uid, PermissionTopic, sessionApprovalConfigKey(sessionID), types.KV{
		"mode": string(mode),
	})
}

// ResolveRunApprovalMode returns session override → user/YAML → manual.
func ResolveRunApprovalMode(ctx context.Context, sessionID string, uid types.Uid) (approval.Mode, error) {
	if sessionID != "" {
		mode, set, err := LoadSessionApprovalMode(ctx, sessionID)
		if err != nil && !errors.Is(err, types.ErrUnavailable) && !errors.Is(err, types.ErrNotFound) {
			return approval.ModeManual, err
		}
		if set {
			return mode, nil
		}
	}
	return LoadUserApprovalMode(ctx, uid)
}
