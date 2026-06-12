package config

const defaultContextWindow = 128000

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

// ContextWindowForModels returns the configured context window for a model name
// using the provided model definitions.
func ContextWindowForModels(models []Model, modelName string) int {
	for _, item := range models {
		if window, ok := item.ContextWindows[modelName]; ok && window > 0 {
			return window
		}
	}
	return defaultContextWindow
}

// ContextWindowForModel returns the configured context window for a model name.
func ContextWindowForModel(modelName string) int {
	return ContextWindowForModels(App.Models, modelName)
}

// MaxContextWindow returns the largest configured context window among the given model names.
func MaxContextWindow(modelNames ...string) int {
	maxWindow := 0
	for _, name := range modelNames {
		if name == "" {
			continue
		}
		window := ContextWindowForModel(name)
		if window > maxWindow {
			maxWindow = window
		}
	}
	if maxWindow == 0 {
		return defaultContextWindow
	}
	return maxWindow
}
