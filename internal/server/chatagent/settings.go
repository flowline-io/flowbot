package chatagent

import (
	"context"
	"fmt"
	"strings"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	agentmodel "github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"

	"github.com/flowline-io/flowbot/internal/store"
)

// SessionSettings holds user-configurable per-session overrides.
type SessionSettings struct {
	// Model overrides chat_agent.chat_model for this session; empty means use the global default.
	Model string `json:"model"`
	// ThinkingLevel overrides the reasoning intensity; empty or "default" uses provider defaults.
	ThinkingLevel string `json:"thinking_level"`
}

// SelectableModel is one entry in the model picker returned to the UI.
type SelectableModel struct {
	// ID is the model identifier registered in flowbot.yaml.
	ID string `json:"id"`
	// Name is a human-readable label (catalog name when available, falls back to ID).
	Name string `json:"name"`
}

// EffectiveSessionSettings is the resolved chat model and thinking level used for a run or UI.
type EffectiveSessionSettings struct {
	// Model is the effective chat model (never empty when chat agent is configured).
	Model string
	// ThinkingLevel is the effective thinking level (never empty; defaults to "default").
	ThinkingLevel string
	// Stored is the raw persisted override pair (may contain empty fields).
	Stored SessionSettings
}

// GetSessionSettings loads per-session overrides, falling back to empty when unset.
func GetSessionSettings(ctx context.Context, sessionID string) (SessionSettings, error) {
	if store.Database == nil {
		return SessionSettings{}, types.ErrUnavailable
	}
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		return SessionSettings{}, err
	}
	return SessionSettings{
		Model:         sess.Model,
		ThinkingLevel: sess.ThinkingLevel,
	}, nil
}

// ResolveEffectiveSessionSettings returns stored overrides plus runtime-resolved values.
func ResolveEffectiveSessionSettings(ctx context.Context, sessionID string) EffectiveSessionSettings {
	stored := SessionSettings{}
	if store.Database != nil && sessionID != "" {
		if sess, err := store.Database.GetChatSession(ctx, sessionID); err == nil && sess != nil {
			stored = SessionSettings{Model: sess.Model, ThinkingLevel: sess.ThinkingLevel}
		} else if err != nil {
			flog.Debug("[chat-agent] resolve settings session=%s: %v", sessionID, err)
		}
	}
	return EffectiveSessionSettings{
		Model:         resolveChatModel(stored.Model),
		ThinkingLevel: resolveThinkingLevel(stored.ThinkingLevel),
		Stored:        stored,
	}
}

// SetSessionSettings persists user-configurable overrides for one session.
func SetSessionSettings(ctx context.Context, sessionID string, s SessionSettings) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	model := strings.TrimSpace(s.Model)
	level := strings.TrimSpace(s.ThinkingLevel)
	if model != "" && !config.ModelRegistered(model) {
		return fmt.Errorf("model %q is not registered: %w", model, types.ErrInvalidArgument)
	}
	if !agentllm.ValidThinkingLevel(level) {
		return fmt.Errorf("invalid thinking_level %q: %w", level, types.ErrInvalidArgument)
	}
	if err := store.Database.UpdateChatSessionSettings(ctx, sessionID, model, level); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] set session settings session=%s: %w", sessionID, err))
		return err
	}
	flog.Debug("[chat-agent] session settings updated session=%s model=%s thinking_level=%s", sessionID, model, level)
	return nil
}

// ResolveSessionChatModel returns the effective chat model for sessionID.
func ResolveSessionChatModel(ctx context.Context, sessionID string) string {
	return ResolveEffectiveSessionSettings(ctx, sessionID).Model
}

// ResolveSessionThinkingLevel returns the effective thinking level for sessionID.
func ResolveSessionThinkingLevel(ctx context.Context, sessionID string) string {
	return ResolveEffectiveSessionSettings(ctx, sessionID).ThinkingLevel
}

func resolveChatModel(stored string) string {
	model := strings.TrimSpace(stored)
	if model != "" && config.ModelRegistered(model) {
		return model
	}
	return config.ChatAgentChatModel()
}

func resolveThinkingLevel(stored string) string {
	level := agentllm.NormalizeThinkingLevel(stored)
	if agentllm.ValidThinkingLevel(level) {
		return level
	}
	return agentllm.ThinkingLevelDefault
}

// BuildSelectableModels returns the model list to show in the UI picker.
// When dual model is enabled, only models sharing the same provider as the
// configured chat_model are included, because chat and tool models must use
// the same provider.
func BuildSelectableModels() []SelectableModel {
	defaultModel := config.ChatAgentChatModel()
	defaultProvider := config.ModelProviderFor(defaultModel)
	filterByProvider := config.App.ChatAgent.ToolModel != "" && defaultProvider != ""

	seen := make(map[string]bool)
	out := make([]SelectableModel, 0)
	for _, group := range config.App.Models {
		if filterByProvider && group.Provider != defaultProvider {
			continue
		}
		for _, name := range group.ModelNames {
			if seen[name] {
				continue
			}
			seen[name] = true
			label := name
			if meta, ok := agentmodel.Lookup(name); ok && meta.Name != "" {
				label = meta.Name
			}
			out = append(out, SelectableModel{ID: name, Name: label})
		}
	}
	return out
}
