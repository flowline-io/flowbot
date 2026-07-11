package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTokenUsageRange(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 12, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name        string
		rangePreset string
		since       string
		until       string
		wantActive  string
		wantLabel   string
		wantErr     bool
	}{
		{
			name:        "default seven day preset",
			rangePreset: "",
			wantActive:  "7d",
			wantLabel:   "Jul 06 - Jul 12",
		},
		{
			name:        "one day preset",
			rangePreset: "1d",
			wantActive:  "1d",
			wantLabel:   "Jul 12 - Jul 12",
		},
		{
			name:        "month to date preset",
			rangePreset: "mtd",
			wantActive:  "mtd",
			wantLabel:   "Jul 01 - Jul 12",
		},
		{
			name:       "custom valid range",
			since:      "2026-07-05",
			until:      "2026-07-11",
			wantActive: "custom",
			wantLabel:  "Jul 05 - Jul 11",
		},
		{
			name:    "custom since after until",
			since:   "2026-07-11",
			until:   "2026-07-05",
			wantErr: true,
		},
		{
			name:    "custom only since",
			since:   "2026-07-05",
			wantErr: true,
		},
		{
			name:        "invalid preset",
			rangePreset: "bad",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			since, until, active, label, err := ResolveTokenUsageRange(tt.rangePreset, tt.since, tt.until, now)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantActive, active)
			assert.Equal(t, tt.wantLabel, label)
			assert.False(t, since.IsZero())
			assert.False(t, until.IsZero())
			assert.False(t, until.Before(since))
		})
	}
}

func TestNormalizeTokenUsageGroupBy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default model", input: "", want: "model"},
		{name: "usage type", input: "usage_type", want: "usage_type"},
		{name: "invalid", input: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeTokenUsageGroupBy(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
