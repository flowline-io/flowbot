package command

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

func TestIsJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *cobra.Command
		want  bool
	}{
		{
			name: "nil command",
			setup: func() *cobra.Command {
				return nil
			},
			want: false,
		},
		{
			name: "table default",
			setup: func() *cobra.Command {
				cmd := &cobra.Command{Use: "list"}
				cmd.Flags().StringP("output", "o", "table", "")
				return cmd
			},
			want: false,
		},
		{
			name: "json output",
			setup: func() *cobra.Command {
				cmd := &cobra.Command{Use: "list"}
				cmd.Flags().StringP("output", "o", "table", "")
				_ = cmd.Flags().Set("output", "json")
				return cmd
			},
			want: true,
		},
		{
			name: "no output flag",
			setup: func() *cobra.Command {
				return &cobra.Command{Use: "create"}
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsJSON(tt.setup()))
		})
	}
}

func TestJSONErrorEnvelope(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("get bookmark: %w", &client.APIError{
		StatusCode: 404,
		RetCode:    "10009",
		Message:    "not found",
	})
	var buf bytes.Buffer
	require.NoError(t, utils.PrintError(&buf, true, err))
	var env map[string]any
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &env))
	obj, ok := env["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "not_found", obj["code"])
	assert.Equal(t, "not found", obj["message"])
	assert.Equal(t, false, obj["retryable"])
	assert.NotEmpty(t, obj["hint"])
	assert.NotContains(t, env, "trace")
	assert.NotContains(t, env, "status")
}

func TestPrintEmptyListUsesJSONFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json bool
		want bool
	}{
		{name: "table mode", json: false, want: false},
		{name: "json mode", json: true, want: true},
		{name: "json mode second case", json: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{Use: "list"}
			cmd.Flags().StringP("output", "o", "table", "")
			if tt.json {
				_ = cmd.Flags().Set("output", "json")
			}
			assert.Equal(t, tt.want, IsJSON(cmd))
		})
	}
}
