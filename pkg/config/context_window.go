package config

import (
	"github.com/flowline-io/flowbot/pkg/agent/model"
)

const (
	defaultReserveTokens    = 10000
	defaultKeepRecentTokens = 20000
)

// CompactionConfig controls session history compaction for the agent.
type CompactionConfig struct {
	// Auto turns on threshold-based compaction before agent runs.
	Auto *bool `json:"auto" yaml:"auto" mapstructure:"auto"`
	// Prune rewrites oversized current tool results before summarization.
	Prune *bool `json:"prune" yaml:"prune" mapstructure:"prune"`
	// Reserved is headroom reserved below the model context window.
	Reserved int `json:"reserved" yaml:"reserved" mapstructure:"reserved"`
	// Enabled preserves compatibility with older configs that used the enabled key.
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" mapstructure:"enabled"`
	// ReserveTokens preserves compatibility with older configs that used reserve_tokens.
	ReserveTokens int `json:"reserve_tokens,omitempty" yaml:"reserve_tokens,omitempty" mapstructure:"reserve_tokens"`
	// KeepRecentTokens controls the approximate token budget kept verbatim after compaction.
	KeepRecentTokens int `json:"keep_recent_tokens,omitempty" yaml:"keep_recent_tokens,omitempty" mapstructure:"keep_recent_tokens"`
}

// WithDefaults fills zero compaction settings with package defaults.
func (c CompactionConfig) WithDefaults() CompactionConfig {
	out := c
	if out.Auto == nil {
		enabled := true
		if out.Enabled != nil {
			enabled = *out.Enabled
		}
		out.Auto = &enabled
	}
	if out.Prune == nil {
		prune := true
		out.Prune = &prune
	}
	if out.Reserved <= 0 {
		if out.ReserveTokens > 0 {
			out.Reserved = out.ReserveTokens
		} else {
			out.Reserved = defaultReserveTokens
		}
	}
	if out.KeepRecentTokens <= 0 {
		out.KeepRecentTokens = defaultKeepRecentTokens
	}
	return out
}

// AutoEnabled reports whether automatic pre-run compaction is enabled.
func (c CompactionConfig) AutoEnabled() bool {
	cfg := c.WithDefaults()
	return cfg.Auto != nil && *cfg.Auto
}

// PruneEnabled reports whether oversized tool results should be rewritten before summarization.
func (c CompactionConfig) PruneEnabled() bool {
	cfg := c.WithDefaults()
	return cfg.Prune != nil && *cfg.Prune
}

// ReservedTokens returns the configured compaction headroom.
func (c CompactionConfig) ReservedTokens() int {
	return c.WithDefaults().Reserved
}

// KeepRecentBudget returns the configured recent-history budget that remains verbatim.
func (c CompactionConfig) KeepRecentBudget() int {
	return c.WithDefaults().KeepRecentTokens
}

// ContextWindowForModels returns the catalog context window for modelName.
//
// Deprecated: models is ignored. Call ContextWindowForModel or ChatAgentContextWindow instead.
func ContextWindowForModels(_ []Model, modelName string) int {
	return model.ContextWindowFor(modelName)
}

// ContextWindowForModel returns the catalog context window for a model name.
func ContextWindowForModel(modelName string) int {
	return model.ContextWindowFor(modelName)
}

// MaxContextWindow returns the largest catalog context window among the given model names.
func MaxContextWindow(modelNames ...string) int {
	return model.MaxContextWindow(modelNames...)
}

// ChatAgentContextWindow returns the effective input budget for the configured chat agent models.
func ChatAgentContextWindow() int {
	chat := ChatAgentChatModel()
	if tool := App.ChatAgent.ToolModel; tool != "" {
		return MaxContextWindow(chat, tool)
	}
	return ContextWindowForModel(chat)
}
