package pipeline

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reloadTestMu serializes tests that mutate package-level reload source/engine.
var reloadTestMu sync.Mutex

func TestReloadDefinitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setup      func(t *testing.T) (DefinitionSource, *Engine)
		wantDefLen int
		wantErr    bool
	}{
		{
			name: "reloads engine from source",
			setup: func(t *testing.T) (DefinitionSource, *Engine) {
				t.Helper()
				engine := NewEngine(nil, nil, nil, nil, nil)
				t.Cleanup(engine.Stop)
				source := func(_ context.Context) ([]Definition, error) {
					return []Definition{
						{Name: "cron-reload", Enabled: true, Trigger: Trigger{Cron: "0 0 * * *"}},
					}, nil
				}
				return source, engine
			},
			wantDefLen: 1,
		},
		{
			name: "no-op when source unset",
			setup: func(t *testing.T) (DefinitionSource, *Engine) {
				t.Helper()
				return nil, nil
			},
			wantDefLen: 0,
		},
		{
			name: "no-op when engine unset",
			setup: func(t *testing.T) (DefinitionSource, *Engine) {
				t.Helper()
				return func(_ context.Context) ([]Definition, error) {
					return []Definition{{Name: "unused"}}, nil
				}, nil
			},
			wantDefLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reloadTestMu.Lock()
			defer reloadTestMu.Unlock()

			SetReloadSource(nil, nil)
			source, engine := tt.setup(t)
			SetReloadSource(source, engine)
			defer SetReloadSource(nil, nil)

			err := ReloadDefinitions(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if engine != nil {
				assert.Len(t, engine.defs, tt.wantDefLen)
			}
		})
	}
}
