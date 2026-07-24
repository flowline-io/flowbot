package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	kvNamespacePrefix = "core/"
	kvInstanceUID     = types.Uid("system")
	kvEnvelopeFlag    = "__core_kv"
)

// KVStore persists namespaced key/value data for core.kv_* ops.
type KVStore interface {
	Get(ctx context.Context, uid types.Uid, namespace, key string) (types.KV, error)
	Set(ctx context.Context, uid types.Uid, namespace, key string, value types.KV) error
	Delete(ctx context.Context, uid types.Uid, namespace, key string) error
}

var (
	kvMu    sync.RWMutex
	kvStore KVStore
)

// SetKVStore wires the persistence backend used by kv_get/set/delete.
func SetKVStore(s KVStore) {
	kvMu.Lock()
	defer kvMu.Unlock()
	kvStore = s
}

func getKVStore() KVStore {
	kvMu.RLock()
	defer kvMu.RUnlock()
	return kvStore
}

func normalizeNamespace(ns string) string {
	ns = strings.TrimSpace(ns)
	ns = strings.TrimPrefix(ns, "/")
	if ns == "" {
		return kvNamespacePrefix + "default"
	}
	if strings.HasPrefix(ns, kvNamespacePrefix) {
		return ns
	}
	return kvNamespacePrefix + ns
}

func resolveKVUID(params map[string]any) types.Uid {
	if raw, ok := params["uid"]; ok {
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return types.Uid(v)
			}
		case types.Uid:
			if !v.IsZero() {
				return v
			}
		}
	}
	return kvInstanceUID
}

func kvGetInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	namespace, err := capability.RequiredString(params, "namespace")
	if err != nil {
		return nil, err
	}
	key, err := capability.RequiredString(params, "key")
	if err != nil {
		return nil, err
	}
	store := getKVStore()
	if store == nil {
		return nil, types.Errorf(types.ErrUnavailable, "kv store is not configured")
	}
	uid := resolveKVUID(params)
	topic := normalizeNamespace(namespace)
	raw, err := store.Get(ctx, uid, topic, key)
	if err != nil {
		return nil, err
	}
	value, expired, err := unwrapKV(raw)
	if err != nil {
		return nil, err
	}
	if expired {
		if delErr := store.Delete(ctx, uid, topic, key); delErr != nil {
			return nil, fmt.Errorf("delete expired kv: %w", delErr)
		}
		return nil, types.Errorf(types.ErrNotFound, "kv key %q expired", key)
	}
	return &capability.InvokeResult{Data: value}, nil
}

func kvSetInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	namespace, err := capability.RequiredString(params, "namespace")
	if err != nil {
		return nil, err
	}
	key, err := capability.RequiredString(params, "key")
	if err != nil {
		return nil, err
	}
	rawValue, ok := params["value"]
	if !ok {
		return nil, types.Errorf(types.ErrInvalidArgument, "value is required")
	}
	store := getKVStore()
	if store == nil {
		return nil, types.Errorf(types.ErrUnavailable, "kv store is not configured")
	}

	var ttlSeconds int64
	if raw, ok := params["ttl_seconds"]; ok && raw != nil {
		switch v := raw.(type) {
		case float64:
			ttlSeconds = int64(v)
		case int:
			ttlSeconds = int64(v)
		case int64:
			ttlSeconds = v
		}
	}

	envelope := wrapKV(rawValue, ttlSeconds)
	uid := resolveKVUID(params)
	topic := normalizeNamespace(namespace)
	if err := store.Set(ctx, uid, topic, key, envelope); err != nil {
		return nil, err
	}
	return &capability.InvokeResult{
		Data: map[string]any{"ok": true, "namespace": topic, "key": key},
		Text: "ok",
	}, nil
}

func kvDeleteInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	namespace, err := capability.RequiredString(params, "namespace")
	if err != nil {
		return nil, err
	}
	key, err := capability.RequiredString(params, "key")
	if err != nil {
		return nil, err
	}
	store := getKVStore()
	if store == nil {
		return nil, types.Errorf(types.ErrUnavailable, "kv store is not configured")
	}
	uid := resolveKVUID(params)
	topic := normalizeNamespace(namespace)
	if err := store.Delete(ctx, uid, topic, key); err != nil {
		return nil, err
	}
	return &capability.InvokeResult{
		Data: map[string]any{"ok": true, "namespace": topic, "key": key},
		Text: "ok",
	}, nil
}

func wrapKV(value any, ttlSeconds int64) types.KV {
	env := types.KV{
		kvEnvelopeFlag: 1,
		"value":        value,
	}
	if ttlSeconds > 0 {
		env["expires_at"] = time.Now().UTC().Add(time.Duration(ttlSeconds) * time.Second).Unix()
	}
	return env
}

func unwrapKV(raw types.KV) (any, bool, error) {
	if raw == nil {
		return nil, false, types.Errorf(types.ErrNotFound, "kv key not found")
	}
	if _, ok := raw[kvEnvelopeFlag]; !ok {
		// Legacy / unexpected shape: return whole map.
		return map[string]any(raw), false, nil
	}
	if exp, ok := raw["expires_at"]; ok && exp != nil {
		var expUnix int64
		switch v := exp.(type) {
		case float64:
			expUnix = int64(v)
		case int64:
			expUnix = v
		case int:
			expUnix = int64(v)
		default:
			return nil, false, fmt.Errorf("invalid expires_at type %T", exp)
		}
		if expUnix > 0 && time.Now().UTC().Unix() >= expUnix {
			return nil, true, nil
		}
	}
	return raw["value"], false, nil
}
