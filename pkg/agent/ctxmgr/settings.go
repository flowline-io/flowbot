package ctxmgr

import "github.com/flowline-io/flowbot/pkg/config"

const (
	defaultReserveTokens    = 10000
	defaultKeepRecentTokens = 20000
)

// Settings controls compaction and branch summarization behavior.
type Settings struct {
	Enabled          bool
	PruneToolOutputs bool
	ReserveTokens    int
	KeepRecentTokens int
}

// WithDefaults fills zero compaction settings.
func (s Settings) WithDefaults() Settings {
	if s.ReserveTokens <= 0 {
		s.ReserveTokens = defaultReserveTokens
	}
	if s.KeepRecentTokens <= 0 {
		s.KeepRecentTokens = defaultKeepRecentTokens
	}
	return s
}

// SettingsFromConfig converts chat agent compaction config into runtime settings.
func SettingsFromConfig(cfg config.CompactionConfig) Settings {
	return Settings{
		Enabled:          cfg.AutoEnabled(),
		PruneToolOutputs: cfg.PruneEnabled(),
		ReserveTokens:    cfg.ReservedTokens(),
		KeepRecentTokens: cfg.KeepRecentBudget(),
	}.WithDefaults()
}
