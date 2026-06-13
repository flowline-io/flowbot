package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextUsagePercent(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		window   int
		reported float64
		want     float64
	}{
		{name: "from token counts", total: 4016, window: 128000, reported: 0, want: 3.1375},
		{name: "fallback when no tokens", total: 0, window: 128000, reported: 12.5, want: 12.5},
		{name: "default window", total: 6400, window: 0, reported: 0, want: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contextUsagePercent(tt.total, tt.window, tt.reported)
			assert.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

func TestFormatContextPercent(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{name: "zero", pct: 0, want: "0%"},
		{name: "small usage", pct: 0.186, want: "0.2%"},
		{name: "single digit", pct: 3.1375, want: "3.1%"},
		{name: "large usage", pct: 42.6, want: "43%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatContextPercent(tt.pct))
		})
	}
}

func TestRenderStatusBarContextPercent(t *testing.T) {
	tests := []struct {
		name    string
		snap    StatusSnapshot
		wantSub string
	}{
		{
			name:    "shows computed percent",
			snap:    StatusSnapshot{Model: "deepseek-v4-flash", TotalTokens: 4016, ContextWindow: 1_048_576},
			wantSub: "0.4%",
		},
		{
			name:    "shows small percent after resume",
			snap:    StatusSnapshot{Model: "deepseek-v4-flash", TotalTokens: 238, ContextWindow: 1_048_576},
			wantSub: "0.0%",
		},
		{
			name:    "progress bar not empty",
			snap:    StatusSnapshot{Model: "test", TotalTokens: 4016, ContextWindow: 128000},
			wantSub: "█",
		},
		{
			name:    "zero tokens stays empty bar",
			snap:    StatusSnapshot{Model: "test", TotalTokens: 0, ContextWindow: 128000},
			wantSub: "0%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderStatusBar(tt.snap, NewStyles())
			assert.Contains(t, got, tt.wantSub)
			if tt.name == "zero tokens stays empty bar" {
				assert.NotContains(t, got, "█")
			}
		})
	}
}

func TestProgressBarFilledWidth(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		want    string
	}{
		{name: "three percent", percent: 3.1375, want: "[█░░░░░░░░░]"},
		{name: "half full", percent: 50, want: "[█████░░░░░]"},
		{name: "full", percent: 100, want: "[██████████]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := progressBar(tt.percent, 10)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, strings.Count(got, "█"), strings.Count(tt.want, "█"))
		})
	}
}
