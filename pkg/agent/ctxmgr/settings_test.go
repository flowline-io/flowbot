package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSettingsFromConfig(t *testing.T) {
	t.Parallel()

	auto := true
	prune := false

	tests := []struct {
		name           string
		cfg            config.CompactionConfig
		wantEnabled    bool
		wantPrune      bool
		wantReserve    int
		wantKeepRecent int
	}{
		{
			name:           "defaults from empty config",
			cfg:            config.CompactionConfig{},
			wantEnabled:    true,
			wantPrune:      true,
			wantReserve:    10000,
			wantKeepRecent: 20000,
		},
		{
			name: "explicit compaction fields",
			cfg: config.CompactionConfig{
				Auto:             &auto,
				Prune:            &prune,
				Reserved:         12000,
				KeepRecentTokens: 8000,
			},
			wantEnabled:    true,
			wantPrune:      false,
			wantReserve:    12000,
			wantKeepRecent: 8000,
		},
		{
			name: "legacy enabled and reserve tokens",
			cfg: config.CompactionConfig{
				Enabled:       new(false),
				ReserveTokens: 4096,
			},
			wantEnabled:    false,
			wantPrune:      true,
			wantReserve:    4096,
			wantKeepRecent: 20000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.SettingsFromConfig(tt.cfg)
			assert.Equal(t, tt.wantEnabled, got.Enabled)
			assert.Equal(t, tt.wantPrune, got.PruneToolOutputs)
			assert.Equal(t, tt.wantReserve, got.ReserveTokens)
			assert.Equal(t, tt.wantKeepRecent, got.KeepRecentTokens)
		})
	}
}
