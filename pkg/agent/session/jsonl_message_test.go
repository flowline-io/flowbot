package session_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalEntryToolResultIsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		wantIsErr bool
		wantFail  bool
	}{
		{
			name: "missing is_error defaults false",
			line: `{"id":"1","type":"message","message":{"role":"toolResult","tool_call_id":"c1","name":"read_file","text":"ok"}}`,
		},
		{
			name:      "is_error true",
			line:      `{"id":"1","type":"message","message":{"role":"toolResult","tool_call_id":"c1","name":"read_file","text":"fail","is_error":true}}`,
			wantIsErr: true,
		},
		{
			name:     "invalid is_error type fails",
			line:     `{"id":"1","type":"message","message":{"role":"toolResult","tool_call_id":"c1","name":"read_file","text":"fail","is_error":"yes"}}`,
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry, err := session.UnmarshalEntry([]byte(tt.line))
			if tt.wantFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			toolResult, ok := entry.Message.(msg.ToolResultMessage)
			require.True(t, ok)
			assert.Equal(t, tt.wantIsErr, toolResult.IsError)
		})
	}
}
