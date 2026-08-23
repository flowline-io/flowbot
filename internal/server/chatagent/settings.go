package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
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
	// ApprovalMode overrides the approval pipeline for this session (manual|auto|off).
	// Empty means inherit the user/YAML default.
	ApprovalMode string `json:"approval_mode,omitempty"`
}

// SelectableModel is one entry in the model picker returned to the UI.
type SelectableModel struct {
	// ID is the model identifier registered in flowbot.yaml.
	ID string `json:"id"`
	// Name is a human-readable label (catalog name when available, falls back to ID).
	Name string `json:"name"`
	// Multimodal is true when the model accepts image, audio, or video input.
	Multimodal bool `json:"multimodal"`
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
	sess, err := store.ChatStoreFromDB().GetChatSession(ctx, sessionID)
	if err != nil {
		return SessionSettings{}, err
	}
	settings := SessionSettings{
		Model:         sess.Model,
		ThinkingLevel: sess.ThinkingLevel,
	}
	mode, set, err := LoadSessionApprovalMode(ctx, sessionID)
	if err != nil {
		return SessionSettings{}, err
	}
	if set {
		settings.ApprovalMode = string(mode)
	}
	return settings, nil
}

// ResolveEffectiveSessionSettings returns stored overrides plus runtime-resolved values.
func ResolveEffectiveSessionSettings(ctx context.Context, sessionID string) EffectiveSessionSettings {
	stored := SessionSettings{}
	if store.Database != nil && sessionID != "" {
		if sess, err := store.ChatStoreFromDB().GetChatSession(ctx, sessionID); err == nil && sess != nil {
			stored = SessionSettings{Model: sess.Model, ThinkingLevel: sess.ThinkingLevel}
		} else if err != nil {
			flog.Debug("[chat-agent] resolve settings session=%s: %v", sessionID, err)
		}
	}
	return EffectiveSessionSettings{
		Model:         resolveChatModel(ctx, stored.Model),
		ThinkingLevel: resolveThinkingLevel(ctx, stored.ThinkingLevel),
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
	if err := store.ChatStoreFromDB().UpdateChatSessionSettings(ctx, sessionID, model, level); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] set session settings session=%s: %w", sessionID, err))
		return err
	}
	if raw := strings.TrimSpace(s.ApprovalMode); raw != "" {
		mode, err := approval.ParseMode(raw)
		if err != nil {
			return fmt.Errorf("invalid approval_mode %q: %w", raw, types.ErrInvalidArgument)
		}
		if err := SaveSessionApprovalMode(ctx, sessionID, mode); err != nil {
			return err
		}
	}
	flog.Debug("[chat-agent] session settings updated session=%s model=%s thinking_level=%s approval_mode=%s",
		sessionID, model, level, strings.TrimSpace(s.ApprovalMode))
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

func resolveChatModel(ctx context.Context, stored string) string {
	model := strings.TrimSpace(stored)
	if model != "" && config.ModelRegistered(model) {
		return model
	}
	return EffectiveChatAgentChatModel(ctx)
}

func resolveThinkingLevel(ctx context.Context, stored string) string {
	level := agentllm.NormalizeThinkingLevel(stored)
	if strings.TrimSpace(stored) != "" && agentllm.ValidThinkingLevel(level) {
		return level
	}
	if serverLevel, ok := effectiveServerThinkingLevel(ctx); ok {
		return serverLevel
	}
	return agentllm.ThinkingLevelDefault
}

// BuildSelectableModels returns the model list to show in the UI picker.
// When dual model is enabled, only models sharing the same provider as the
// configured chat_model are included, because chat and tool models must use
// the same provider.
func BuildSelectableModels() []SelectableModel {
	return BuildSelectableModelsWithContext(context.Background())
}

// BuildSelectableModelsWithContext returns the model picker list using effective server defaults.
func BuildSelectableModelsWithContext(ctx context.Context) []SelectableModel {
	defaultModel := EffectiveChatAgentChatModel(ctx)
	_, toolModel, _, err := ResolveEffectiveChatAgentModels(ctx)
	if err != nil {
		toolModel = config.App.ChatAgent.ToolModel
	}
	defaultProvider := config.ModelProviderFor(defaultModel)
	filterByProvider := toolModel != "" && defaultProvider != ""

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
			out = append(out, SelectableModel{
				ID:         name,
				Name:       label,
				Multimodal: agentmodel.AcceptsMediaInput(name),
			})
		}
	}
	return out
}
