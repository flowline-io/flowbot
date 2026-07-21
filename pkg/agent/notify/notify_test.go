package notify

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	pkgnotify "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestSendToolExecute(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		wantError  bool
		wantSubstr string
	}{
		{
			name:       "empty message",
			args:       map[string]any{"message": "  "},
			wantError:  true,
			wantSubstr: "message is required",
		},
		{
			name:       "nil message",
			args:       map[string]any{},
			wantError:  true,
			wantSubstr: "message is required",
		},
		{
			name:       "missing store defaults",
			args:       map[string]any{"message": "hello"},
			wantError:  true,
			wantSubstr: "unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := (SendTool{}).Execute(context.Background(), "c1", tt.args, nil)
			require.NoError(t, err)
			text := toolResultText(res)
			assert.True(t, res.IsError)
			assert.Contains(t, text, tt.wantSubstr)
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		reg     *tool.Registry
		wantErr bool
	}{
		{name: "nil registry", reg: nil, wantErr: true},
		{name: "ok", reg: tool.NewRegistry(), wantErr: false},
		{name: "active names", reg: tool.NewRegistry(), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.reg, types.Uid("u1"))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{SendToolName}, ActiveToolNames())
		})
	}
}

func TestSendErrorResult(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{name: "no channel", err: pkgnotify.ErrNoDefaultChannel, wantSubstr: "default notification channel"},
		{name: "no template", err: pkgnotify.ErrNoDefaultTemplate, wantSubstr: "default notification template"},
		{name: "not found", err: types.ErrNotFound, wantSubstr: "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendErrorResult("c1", SendToolName, tt.err)
			assert.True(t, res.IsError)
			assert.Contains(t, toolResultText(res), tt.wantSubstr)
		})
	}
}

func toolResultText(res msg.ToolResultMessage) string {
	var b strings.Builder
	for _, p := range res.Parts {
		if tp, ok := p.(msg.TextPart); ok {
			b.WriteString(tp.Text)
		}
	}
	return b.String()
}
