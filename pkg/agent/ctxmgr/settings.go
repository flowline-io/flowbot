package ctxmgr

import "github.com/flowline-io/flowbot/pkg/config"

const (
	defaultReserveTokens    = 16384
	defaultKeepRecentTokens = 20000
)

// Settings controls compaction and branch summarization behavior.
type Settings struct {
	Enabled          bool
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
		Enabled:          cfg.Enabled,
		ReserveTokens:    cfg.ReserveTokens,
		KeepRecentTokens: cfg.KeepRecentTokens,
	}.WithDefaults()
}
