package llm_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestFunctionTool_Info(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tool     *llm.FunctionTool
		wantName string
		wantErr  bool
	}{
		{
			name:     "full tool metadata",
			tool:     &llm.FunctionTool{Name: "my_tool", Description: "does something"},
			wantName: "my_tool",
		},
		{
			name:    "empty name tool",
			tool:    &llm.FunctionTool{Name: "", Description: "no name"},
			wantErr: false,
		},
		{
			name:     "nil parameters",
			tool:     &llm.FunctionTool{Name: "no_params", Description: "without params"},
			wantName: "no_params",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info, err := tt.tool.Info(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
		})
	}
}

func TestFunctionTool_InvokableRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tool      *llm.FunctionTool
		input     string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "normal execution echoes input",
			tool: &llm.FunctionTool{
				Name: "echo",
				Execute: func(ctx context.Context, input string) (string, error) {
					return input, nil
				},
			},
			input: "hello",
			want:  "hello",
		},
		{
			name: "execution error propagates",
			tool: &llm.FunctionTool{
				Name: "failing",
				Execute: func(ctx context.Context, input string) (string, error) {
					return "", errors.New("tool failed")
				},
			},
			input:     "input",
			wantErr:   true,
			errSubstr: "tool failed",
		},
		{
			name: "executes with empty input",
			tool: &llm.FunctionTool{
				Name: "len",
				Execute: func(ctx context.Context, input string) (string, error) {
					return fmt.Sprintf("%d", len(input)), nil
				},
			},
			input: "",
			want:  "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.tool.InvokableRun(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{name: "valid json", input: `{"key":"value"}`, wantErr: false, wantLen: 1},
		{name: "empty string", input: "", wantErr: false, wantLen: 0},
		{name: "invalid json", input: `{invalid}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := llm.ConvertFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}
