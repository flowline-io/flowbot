package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// ServerConfigUID is the ConfigData owner for chat agent server-wide defaults.
	ServerConfigUID types.Uid = "_server"
	// ServerDefaultsKey is the ConfigData key for server-wide chat agent defaults.
	ServerDefaultsKey = "server_defaults"
	// ServerDefaultFormInherit means the field inherits from flowbot.yaml.
	ServerDefaultFormInherit = permission.FormActionInherit
	// ServerDefaultToolNone explicitly disables dual-model routing for tool_model.
	ServerDefaultToolNone = "__none__"
)

const serverDefaultsCacheTTL = 5 * time.Second

type serverDefaultsCacheEntry struct {
	defaults  ServerDefaults
	expiresAt time.Time
}

var (
	serverDefaultsCache   serverDefaultsCacheEntry
	serverDefaultsCacheMu sync.Mutex
	serverDefaultsCached  bool
)

// ServerDefaults holds optional server-wide overrides stored in ConfigData.
type ServerDefaults struct {
	ChatModelSet     bool
	ChatModel        string
	ToolModelSet     bool
	ToolModel        string
	ThinkingLevelSet bool
	ThinkingLevel    string
}

// ServerDefaultsFormInput is the submitted permissions form payload for server defaults.
type ServerDefaultsFormInput struct {
	ChatModel     string
	ToolModel     string
	ThinkingLevel string
}

// LoadServerDefaults reads stored server-wide overrides. Missing record returns zero defaults.
func LoadServerDefaults(ctx context.Context) (ServerDefaults, error) {
	if cached, ok := loadServerDefaultsCache(); ok {
		return cached, nil
	}
	if store.Database == nil {
		return ServerDefaults{}, types.ErrUnavailable
	}
	raw, err := store.ModuleDataStoreFromDB().ConfigGet(ctx, ServerConfigUID, PermissionTopic, ServerDefaultsKey)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			storeServerDefaultsCache(ServerDefaults{})
			return ServerDefaults{}, nil
		}
		return ServerDefaults{}, fmt.Errorf("load server defaults: %w", err)
	}
	out := parseServerDefaultsKV(raw)
	storeServerDefaultsCache(out)
	return out, nil
}

// SaveServerDefaults persists server-wide overrides from the permissions form.
func SaveServerDefaults(ctx context.Context, input ServerDefaultsFormInput) error {
	next, err := serverDefaultsFromForm(input)
	if err != nil {
		return err
	}
	if err := validateServerDefaults(next); err != nil {
		return err
	}
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if !next.anySet() {
		if err := store.ModuleDataStoreFromDB().ConfigDelete(ctx, ServerConfigUID, PermissionTopic, ServerDefaultsKey); err != nil {
			if errors.Is(err, types.ErrNotFound) {
				invalidateServerDefaultsCache()
				return nil
			}
			return err
		}
		invalidateServerDefaultsCache()
		return nil
	}
	if err := store.ModuleDataStoreFromDB().ConfigSet(ctx, ServerConfigUID, PermissionTopic, ServerDefaultsKey, encodeServerDefaultsKV(next)); err != nil {
		return err
	}
	storeServerDefaultsCache(next)
	return nil
}

// DeleteServerDefaults removes all server-wide overrides so YAML applies.
func DeleteServerDefaults(ctx context.Context) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	if err := store.ModuleDataStoreFromDB().ConfigDelete(ctx, ServerConfigUID, PermissionTopic, ServerDefaultsKey); err != nil {
		if errors.Is(err, types.ErrNotFound) {
			invalidateServerDefaultsCache()
			return nil
		}
		return err
	}
	invalidateServerDefaultsCache()
	return nil
}

// EffectiveChatAgentChatModel returns DB override when set, otherwise YAML chat_model.
func EffectiveChatAgentChatModel(ctx context.Context) string {
	stored, err := LoadServerDefaults(ctx)
	if err == nil && stored.ChatModelSet {
		return stored.ChatModel
	}
	return config.ChatAgentChatModel()
}

// ResolveEffectiveChatAgentModels resolves effective chat/tool models with server DB overrides.
func ResolveEffectiveChatAgentModels(ctx context.Context) (chat, tool string, dual bool, err error) {
	chat = config.ChatAgentChatModel()
	tool = config.App.ChatAgent.ToolModel
	stored, err := LoadServerDefaults(ctx)
	if err != nil && !errors.Is(err, types.ErrUnavailable) {
		return "", "", false, err
	}
	if err == nil {
		if stored.ChatModelSet {
			chat = stored.ChatModel
		}
		if stored.ToolModelSet {
			tool = stored.ToolModel
		}
	}
	return config.ResolveChatAgentModelPair(chat, tool)
}

func serverDefaultsFromForm(input ServerDefaultsFormInput) (ServerDefaults, error) {
	next := ServerDefaults{}
	if err := applyServerDefaultField(&next.ChatModelSet, &next.ChatModel, input.ChatModel, true); err != nil {
		return ServerDefaults{}, err
	}
	if err := applyServerDefaultToolField(&next, input.ToolModel); err != nil {
		return ServerDefaults{}, err
	}
	if err := applyServerDefaultField(&next.ThinkingLevelSet, &next.ThinkingLevel, input.ThinkingLevel, false); err != nil {
		return ServerDefaults{}, err
	}
	return next, nil
}

func effectiveServerThinkingLevel(ctx context.Context) (string, bool) {
	stored, err := LoadServerDefaults(ctx)
	if err != nil || !stored.ThinkingLevelSet {
		return "", false
	}
	level := agentllm.NormalizeThinkingLevel(stored.ThinkingLevel)
	if !agentllm.ValidThinkingLevel(level) {
		return "", false
	}
	return level, true
}

func applyServerDefaultField(set *bool, value *string, raw string, requireModel bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == ServerDefaultFormInherit {
		*set = false
		*value = ""
		return nil
	}
	if requireModel {
		if !config.ModelRegistered(raw) {
			return fmt.Errorf("model %q is not registered: %w", raw, types.ErrInvalidArgument)
		}
	} else if !agentllm.ValidThinkingLevel(agentllm.NormalizeThinkingLevel(raw)) {
		return fmt.Errorf("invalid thinking_level %q: %w", raw, types.ErrInvalidArgument)
	}
	*set = true
	*value = raw
	return nil
}

func applyServerDefaultToolField(next *ServerDefaults, raw string) error {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "", ServerDefaultFormInherit:
		next.ToolModelSet = false
		next.ToolModel = ""
		return nil
	case ServerDefaultToolNone:
		next.ToolModelSet = true
		next.ToolModel = ""
		return nil
	default:
		if !config.ModelRegistered(raw) {
			return fmt.Errorf("model %q is not registered: %w", raw, types.ErrInvalidArgument)
		}
		next.ToolModelSet = true
		next.ToolModel = raw
		return nil
	}
}

func validateServerDefaults(next ServerDefaults) error {
	chat := config.ChatAgentChatModel()
	tool := config.App.ChatAgent.ToolModel
	if next.ChatModelSet {
		chat = next.ChatModel
	}
	if next.ToolModelSet {
		tool = next.ToolModel
	}
	_, _, _, err := config.ResolveChatAgentModelPair(chat, tool)
	return err
}

func (s ServerDefaults) anySet() bool {
	return s.ChatModelSet || s.ToolModelSet || s.ThinkingLevelSet
}

func parseServerDefaultsKV(raw types.KV) ServerDefaults {
	out := ServerDefaults{}
	overrideList, _ := raw.List("overrides")
	keys := make([]string, 0, len(overrideList))
	for _, item := range overrideList {
		if key, ok := item.(string); ok {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		switch key {
		case "chat_model":
			if model, ok := raw.String("chat_model"); ok {
				out.ChatModelSet = true
				out.ChatModel = strings.TrimSpace(model)
			}
		case "tool_model":
			out.ToolModelSet = true
			if model, ok := raw.String("tool_model"); ok {
				out.ToolModel = strings.TrimSpace(model)
			}
		case "thinking_level":
			if level, ok := raw.String("thinking_level"); ok {
				out.ThinkingLevelSet = true
				out.ThinkingLevel = strings.TrimSpace(level)
			}
		}
	}
	return out
}

func encodeServerDefaultsKV(stored ServerDefaults) types.KV {
	overrides := make([]string, 0, 3)
	kv := types.KV{}
	if stored.ChatModelSet {
		overrides = append(overrides, "chat_model")
		kv["chat_model"] = stored.ChatModel
	}
	if stored.ToolModelSet {
		overrides = append(overrides, "tool_model")
		kv["tool_model"] = stored.ToolModel
	}
	if stored.ThinkingLevelSet {
		overrides = append(overrides, "thinking_level")
		kv["thinking_level"] = stored.ThinkingLevel
	}
	kv["overrides"] = overrides
	return kv
}

func loadServerDefaultsCache() (ServerDefaults, bool) {
	serverDefaultsCacheMu.Lock()
	defer serverDefaultsCacheMu.Unlock()
	if !serverDefaultsCached || time.Now().After(serverDefaultsCache.expiresAt) {
		serverDefaultsCached = false
		return ServerDefaults{}, false
	}
	return serverDefaultsCache.defaults, true
}

func storeServerDefaultsCache(defaults ServerDefaults) {
	serverDefaultsCacheMu.Lock()
	defer serverDefaultsCacheMu.Unlock()
	serverDefaultsCache = serverDefaultsCacheEntry{
		defaults:  defaults,
		expiresAt: time.Now().Add(serverDefaultsCacheTTL),
	}
	serverDefaultsCached = true
}

func invalidateServerDefaultsCache() {
	serverDefaultsCacheMu.Lock()
	defer serverDefaultsCacheMu.Unlock()
	serverDefaultsCached = false
}

// ResetServerDefaultsCacheForTest clears the in-memory server defaults cache.
func ResetServerDefaultsCacheForTest() {
	invalidateServerDefaultsCache()
}
