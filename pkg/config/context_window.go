package config

import (
	"github.com/flowline-io/flowbot/pkg/agent/model"
)

const defaultContextWindow = model.DefaultContextWindow

const (
	defaultReserveTokens    = 16384
	defaultKeepRecentTokens = 20000
)

// CompactionConfig controls automatic session history compaction for the chat agent.
type CompactionConfig struct {
	// Enabled turns on threshold-based compaction before agent runs.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// ReserveTokens is headroom reserved below the model context window.
	ReserveTokens int `json:"reserve_tokens" yaml:"reserve_tokens" mapstructure:"reserve_tokens"`
	// KeepRecentTokens is the approximate token budget kept verbatim after compaction.
	KeepRecentTokens int `json:"keep_recent_tokens" yaml:"keep_recent_tokens" mapstructure:"keep_recent_tokens"`
}

// WithDefaults fills zero compaction settings with package defaults.
func (c CompactionConfig) WithDefaults() CompactionConfig {
	if c.ReserveTokens <= 0 {
		c.ReserveTokens = defaultReserveTokens
	}
	if c.KeepRecentTokens <= 0 {
		c.KeepRecentTokens = defaultKeepRecentTokens
	}
	return c
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
