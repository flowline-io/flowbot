package agent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRunner struct {
	reply     string
	sessionID string
	err       error
	last      RunParams
}

func (s *stubRunner) Run(_ context.Context, params RunParams) (*RunResult, error) {
	s.last = params
	if s.err != nil {
		return nil, s.err
	}
	return &RunResult{Reply: s.reply, SessionID: s.sessionID}, nil
}

func TestRunInvokerRequiresPrompt(t *testing.T) {
	SetRunner(&stubRunner{reply: "ok"})
	t.Cleanup(func() { SetRunner(nil) })

	tests := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing prompt", params: map[string]any{}},
		{name: "empty prompt", params: map[string]any{"prompt": ""}},
		{name: "nil prompt value", params: map[string]any{"prompt": nil}},
		{name: "whitespace prompt", params: map[string]any{"prompt": "   "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runInvoker(context.Background(), tt.params)
			require.Error(t, err)
		})
	}
}

func TestRunInvokerRequiresRunner(t *testing.T) {
	SetRunner(nil)

	_, err := runInvoker(context.Background(), map[string]any{"prompt": "hello"})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrUnavailable)
}

func TestRunInvokerReturnsReply(t *testing.T) {
	stub := &stubRunner{reply: "agent says hi", sessionID: "sess-99"}
	SetRunner(stub)
	t.Cleanup(func() { SetRunner(nil) })

	res, err := runInvoker(context.Background(), map[string]any{
		"prompt": "summarize",
		"uid":    "user-42",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	data, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "agent says hi", data["reply"])
	assert.Equal(t, "sess-99", data["session_id"])
	assert.Equal(t, "agent says hi", res.Text)
	assert.Equal(t, types.Uid("user-42"), stub.last.UID)
}

func TestRunInvokerParsesOptionalUID(t *testing.T) {
	stub := &stubRunner{reply: "ok", sessionID: "sess-1"}
	SetRunner(stub)
	t.Cleanup(func() { SetRunner(nil) })

	_, err := runInvoker(context.Background(), map[string]any{"prompt": "go"})
	require.NoError(t, err)
	assert.True(t, stub.last.UID.IsZero())
}

func TestRunInvokerParsesToolsAndSkills(t *testing.T) {
	stub := &stubRunner{reply: "ok", sessionID: "sess-1"}
	SetRunner(stub)
	t.Cleanup(func() { SetRunner(nil) })

	tests := []struct {
		name   string
		params map[string]any
		tools  []string
		skills []string
	}{
		{
			name: "string slices are forwarded",
			params: map[string]any{
				"prompt": "go",
				"tools":  []string{"read_file"},
				"skills": []string{"skill-a"},
			},
			tools:  []string{"read_file"},
			skills: []string{"skill-a"},
		},
		{
			name: "any slices are forwarded",
			params: map[string]any{
				"prompt": "go",
				"tools":  []any{"web_search"},
				"skills": []any{"skill-b"},
			},
			tools:  []string{"web_search"},
			skills: []string{"skill-b"},
		},
		{
			name: "empty arrays are omitted",
			params: map[string]any{
				"prompt": "go",
				"tools":  []string{},
				"skills": []any{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runInvoker(context.Background(), tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.tools, stub.last.Tools)
			assert.Equal(t, tt.skills, stub.last.Skills)
		})
	}
}

func TestRunInvokerRejectsInvalidToolsType(t *testing.T) {
	SetRunner(&stubRunner{reply: "ok"})
	t.Cleanup(func() { SetRunner(nil) })

	_, err := runInvoker(context.Background(), map[string]any{
		"prompt": "go",
		"tools":  "read_file",
	})
	require.Error(t, err)
}

func TestHealthInvoker(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() {
		config.App.ChatAgent = prev
		SetRunner(nil)
	})

	tests := []struct {
		name      string
		chatModel string
		runner    Runner
		wantOK    bool
		wantErr   bool
	}{
		{name: "healthy with runner", runner: &stubRunner{}, wantOK: true},
		{name: "healthy with chat model", chatModel: "gpt-test", wantOK: true},
		{name: "unconfigured", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.ChatModel = tt.chatModel
			SetRunner(tt.runner)
			res, err := healthInvoker(context.Background(), nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			ok, isBool := res.Data.(bool)
			require.True(t, isBool)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
