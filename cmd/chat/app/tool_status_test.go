package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestFormatToolEventLine(t *testing.T) {
	tests := []struct {
		name string
		ev   client.ChatStreamEvent
		want string
	}{
		{
			name: "top-level tool start",
			ev:   client.ChatStreamEvent{Name: "run_terminal"},
			want: "⚙ Running tool: run_terminal...",
		},
		{
			name: "top-level tool update",
			ev:   client.ChatStreamEvent{Name: "run_terminal", Stdout: "fetching"},
			want: "✓ run_terminal: fetching",
		},
		{
			name: "task delegation start",
			ev:   client.ChatStreamEvent{Name: "task", Subagent: "general-purpose"},
			want: "⤷ Delegating to subagent: general-purpose...",
		},
		{
			name: "legacy task delegation name",
			ev:   client.ChatStreamEvent{Name: "task (general-purpose)"},
			want: "⤷ Delegating to subagent: general-purpose...",
		},
		{
			name: "subagent inner tool start",
			ev:   client.ChatStreamEvent{Name: "web_search", Subagent: "general-purpose"},
			want: "  ↳ ⚙ Running tool: web_search...",
		},
		{
			name: "subagent inner tool update",
			ev: client.ChatStreamEvent{
				Name:     "web_search",
				Subagent: "general-purpose",
				Stdout:   "searching...",
			},
			want: "  ↳ ✓ web_search: searching...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatToolEventLine(tt.ev))
		})
	}
}

func TestHandleStreamEventSubagentTool(t *testing.T) {
	tests := []struct {
		name       string
		events     []client.ChatStreamEvent
		wantSubstr []string
	}{
		{
			name: "shows delegation then inner tool progress",
			events: []client.ChatStreamEvent{
				{Type: "tool", Name: "task", Subagent: "general-purpose"},
				{Type: "tool", Name: "web_search", Subagent: "general-purpose"},
				{Type: "tool", Name: "web_search", Subagent: "general-purpose", Stdout: "searching..."},
			},
			wantSubstr: []string{
				"Delegating to subagent: general-purpose",
				"Running tool: web_search",
				"web_search: searching",
			},
		},
		{
			name: "top-level tool unchanged",
			events: []client.ChatStreamEvent{
				{Type: "tool", Name: "echo"},
				{Type: "tool", Name: "echo", Stdout: "pong"},
			},
			wantSubstr: []string{"Running tool: echo", "echo: pong"},
		},
		{
			name: "legacy task name still delegates",
			events: []client.ChatStreamEvent{
				{Type: "tool", Name: "task (general-purpose)"},
			},
			wantSubstr: []string{"Delegating to subagent: general-purpose"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &Model{phase: PhaseStreaming, width: 80, styles: NewStyles()}
			for _, ev := range tt.events {
				m, _ = m.handleStreamEvent(ev)
			}
			out := stripANSI(m.stream.overlay.String())
			for _, want := range tt.wantSubstr {
				assert.Contains(t, out, want)
			}
		})
	}
}
