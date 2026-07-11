package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEnabledInYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "empty defaults to enabled",
			yaml: "",
			want: true,
		},
		{
			name: "missing enabled key defaults to enabled",
			yaml: "name: test\ntriggers: []\nsteps: []",
			want: true,
		},
		{
			name: "explicit true",
			yaml: "name: test\nenabled: true\nsteps: []",
			want: true,
		},
		{
			name: "explicit false",
			yaml: "name: test\nenabled: false\nsteps: []",
			want: false,
		},
		{
			name: "invalid yaml defaults to enabled",
			yaml: "name: [\nenabled: true",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEnabledInYAML(tt.yaml)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetEnabledInYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		enabled bool
		wantErr bool
	}{
		{
			name:    "empty input rejected",
			yaml:    "",
			enabled: false,
			wantErr: true,
		},
		{
			name:    "invalid yaml rejected",
			yaml:    "name: [",
			enabled: false,
			wantErr: true,
		},
		{
			name:    "pause published definition",
			yaml:    "name: test\nenabled: true\ntriggers: []\nsteps: []",
			enabled: false,
		},
		{
			name:    "resume paused definition",
			yaml:    "name: test\nenabled: false\ntriggers: []\nsteps: []",
			enabled: true,
		},
		{
			name: "pause disables cron trigger",
			yaml: `name: test
enabled: true
triggers:
  - type: event
    enabled: true
    event: item.created
  - type: cron
    enabled: true
    cron: "0 * * * *"
steps: []`,
			enabled: false,
		},
		{
			name: "resume enables cron trigger",
			yaml: `name: test
enabled: false
triggers:
  - type: cron
    enabled: false
    cron: "0 * * * *"
steps: []`,
			enabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetEnabledInYAML(tt.yaml, tt.enabled)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.enabled, IsEnabledInYAML(got))

			def, parseErr := ParseEditorYAML(got)
			require.NoError(t, parseErr)
			for _, trigger := range def.Triggers {
				if trigger.Type != "cron" {
					continue
				}
				assert.Equal(t, tt.enabled, trigger.Enabled)
			}
		})
	}
}

func TestExpandDefinitionsPausedCron(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantCount int
	}{
		{
			name: "active cron trigger expands",
			yaml: `name: cron-pl
enabled: true
triggers:
  - type: cron
    enabled: true
    cron: "0 * * * *"
steps: []`,
			wantCount: 1,
		},
		{
			name: "paused pipeline skips cron",
			yaml: `name: cron-pl
enabled: false
triggers:
  - type: cron
    enabled: false
    cron: "0 * * * *"
steps: []`,
			wantCount: 0,
		},
		{
			name: "active pipeline with disabled cron trigger skips cron",
			yaml: `name: cron-pl
enabled: true
triggers:
  - type: cron
    enabled: false
    cron: "0 * * * *"
steps: []`,
			wantCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseEditorYAML(tt.yaml)
			require.NoError(t, err)
			got := ExpandDefinitions([]EditorDefinition{*def})
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func TestEngine_ReloadDropsPausedCron(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)

	active := []Definition{{
		Name: "cron-pl", Enabled: true, Trigger: Trigger{Cron: "0 * * * *"},
	}}
	paused := []Definition{}

	e := NewEngineWithClock(active, nil, nil, noopPC, noopEC, clock)
	defer e.Stop()
	require.Len(t, e.cron.Entries(), 1)

	require.NoError(t, e.Reload(paused))
	assert.Empty(t, e.cron.Entries())
}
