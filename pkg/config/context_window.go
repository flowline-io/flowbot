package config

import (
	"github.com/flowline-io/flowbot/pkg/agent/model"
)

const defaultContextWindow = model.DefaultContextWindow

const (
	defaultReserveTokens    = 10000
	defaultKeepRecentTokens = 20000
)

// CompactionConfig controls session history compaction for the chat agent.
type CompactionConfig struct {
	// Auto turns on threshold-based compaction before agent runs.
	Auto *bool `json:"auto" yaml:"auto" mapstructure:"auto"`
	// Prune removes older tool outputs from the compaction prompt before summarization.
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
	if c.Auto == nil {
		enabled := true
		if c.Enabled != nil {
			enabled = *c.Enabled
		}
		c.Auto = &enabled
	}
	if c.Prune == nil {
		prune := true
		c.Prune = &prune
	}
	if c.Reserved <= 0 {
		if c.ReserveTokens > 0 {
			c.Reserved = c.ReserveTokens
		} else {
			c.Reserved = defaultReserveTokens
		}
	}
	if c.KeepRecentTokens <= 0 {
		c.KeepRecentTokens = defaultKeepRecentTokens
	}
	return c
}

// AutoEnabled reports whether automatic pre-run compaction is enabled.
func (c CompactionConfig) AutoEnabled() bool {
	cfg := c.WithDefaults()
	return cfg.Auto != nil && *cfg.Auto
}

// PruneEnabled reports whether older tool results should be pruned during compaction.
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
