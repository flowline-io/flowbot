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

// ContextWindowForModel returns the configured context window for a model name.
func ContextWindowForModel(modelName string) int {
	for _, item := range App.Models {
		if window, ok := item.ContextWindows[modelName]; ok && window > 0 {
			return window
		}
	}
	return defaultContextWindow
}
