package ctxmgr_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/stretchr/testify/assert"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		msg  msg.AgentMessage
		want int
	}{
		{name: "short user", msg: msg.NewUserMessage("hello"), want: 2},
		{name: "assistant text", msg: msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "12345678"}}}, want: 2},
		{name: "branch summary", msg: msg.BranchSummaryMessage{Summary: "1234"}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ctxmgr.EstimateTokens(tt.msg))
		})
	}
}

func TestEstimateContextTokens(t *testing.T) {
	tests := []struct {
		name string
		msgs []msg.AgentMessage
		want int
	}{
		{
			name: "usage plus trailing",
			msgs: []msg.AgentMessage{
				msg.NewUserMessage("a"),
				msg.AssistantMessage{
					Parts: []msg.ContentPart{msg.TextPart{Text: "b"}},
					Usage: &msg.Usage{TotalTokens: 100},
				},
				msg.NewUserMessage("cccc"),
			},
			want: 101,
		},
		{
			name: "no usage heuristic",
			msgs: []msg.AgentMessage{
				msg.NewUserMessage("12345678"),
			},
			want: 2,
		},
		{
			name: "empty",
			msgs: nil,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.EstimateContextTokens(tt.msgs)
			assert.Equal(t, tt.want, got.Tokens)
		})
	}
}

func TestShouldCompact(t *testing.T) {
	tests := []struct {
		name    string
		tokens  int
		window  int
		enabled bool
		want    bool
	}{
		{name: "below threshold", tokens: 1000, window: 128000, enabled: true, want: false},
		{name: "above threshold", tokens: 120000, window: 128000, enabled: true, want: true},
		{name: "disabled", tokens: 120000, window: 128000, enabled: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.ShouldCompact(tt.tokens, tt.window, ctxmgr.Settings{Enabled: tt.enabled, ReserveTokens: 16384, KeepRecentTokens: 20000})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsContextOverflowErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "generic error", err: errString("database unavailable"), want: false},
		{name: "openai overflow", err: errString("Your input exceeds the context window of this model"), want: true},
		{name: "rate limit excluded", err: errString("rate limit exceeded for tenant"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ctxmgr.IsContextOverflowErr(tt.err))
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
