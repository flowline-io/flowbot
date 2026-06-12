package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestEstimateHistoryTokens(t *testing.T) {
	tests := []struct {
		name string
		msgs []client.ChatHistoryMessage
		want int
	}{
		{name: "empty history", msgs: nil, want: 0},
		{name: "single message", msgs: []client.ChatHistoryMessage{{Text: string(make([]byte, 400))}}, want: 100},
		{name: "multiple messages", msgs: []client.ChatHistoryMessage{
			{Text: "hello"},
			{Text: string(make([]byte, 396))},
		}, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EstimateHistoryTokens(tt.msgs))
		})
	}
}

func TestApplyHistoryUsage(t *testing.T) {
	tests := []struct {
		name       string
		tokens     int
		window     int
		wantTokens int
		wantPct    float64
	}{
		{name: "restores usage on resume", tokens: 4016, window: 128000, wantTokens: 4016, wantPct: 3.1375},
		{name: "defaults window", tokens: 6400, window: 0, wantTokens: 6400, wantPct: 5},
		{name: "clears to zero", tokens: 0, window: 128000, wantTokens: 0, wantPct: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.status.ContextWindow = tt.window
			m.applyHistoryUsage(tt.tokens)
			assert.Equal(t, tt.wantTokens, m.status.TotalTokens)
			assert.InDelta(t, tt.wantPct, m.status.ContextPercent, 0.0001)
		})
	}
}

func TestResetSessionUsage(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "clears token counters"},
		{name: "idempotent reset"},
		{name: "safe on fresh model"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.status.TotalTokens = 5000
			m.status.ContextPercent = 12.5
			m.resetSessionUsage()
			assert.Zero(t, m.status.TotalTokens)
			assert.Zero(t, m.status.ContextPercent)
		})
	}
}
