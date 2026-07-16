package ctxmgr_test

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapOverflowErrorAndResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantCode string
	}{
		{name: "nil stays nil", err: nil, wantNil: true},
		{name: "overflow wraps", err: errors.New("context length exceeded"), wantCode: "overflow"},
		{name: "non-overflow unchanged", err: errors.New("connection refused")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ctxmgr.WrapOverflowError(tt.err)
			if tt.wantNil {
				assert.NoError(t, got)
				return
			}
			if tt.wantCode != "" {
				require.Error(t, got)
				assert.Equal(t, tt.wantCode, result.CodeOf(got))
				return
			}
			assert.Equal(t, tt.err, got)
		})
	}
}

func TestIsOverflowResult(t *testing.T) {
	t.Parallel()

	overflowMsg := agent.AssistantMessage{
		StopReason: "error",
		Parts:      []agent.ContentPart{agent.TextPart{Text: "maximum context length is 128000 tokens"}},
	}

	tests := []struct {
		name     string
		err      error
		messages []agent.AgentMessage
		window   int
		want     bool
	}{
		{name: "stream error overflow", err: result.NewOverflowError("overflow", nil), want: true},
		{name: "assistant message overflow", messages: []agent.AgentMessage{overflowMsg}, window: 128000, want: true},
		{name: "no overflow signals", messages: []agent.AgentMessage{agent.NewUserMessage("hi")}, window: 128000, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ctxmgr.IsOverflowResult(tt.err, tt.messages, tt.window))
		})
	}
}
