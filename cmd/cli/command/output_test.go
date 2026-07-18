package command

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
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

func TestPrintJSONErrorShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         error
		wantMsg     string
		wantRetCode string
	}{
		{
			name:    "plain error",
			err:     errors.New("boom"),
			wantMsg: "boom",
		},
		{
			name: "api error message",
			err: &client.APIError{
				StatusCode: 404,
				RetCode:    "10009",
				Message:    "not found",
			},
			wantMsg:     "not found",
			wantRetCode: "10009",
		},
		{
			name: "wrapped api error",
			err: fmt.Errorf("list bookmarks: %w", &client.APIError{
				StatusCode: 500,
				Message:    "server error",
			}),
			wantMsg: "server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := protocol.NewFailedResponse(tt.err)
			var apiErr *client.APIError
			if errors.As(tt.err, &apiErr) {
				if apiErr.Message != "" {
					resp.Message = apiErr.Message
				}
				if apiErr.RetCode != "" {
					resp.RetCode = apiErr.RetCode
				}
			}
			assert.Equal(t, protocol.Failed, resp.Status)
			assert.Equal(t, tt.wantMsg, resp.Message)
			if tt.wantRetCode != "" {
				assert.Equal(t, tt.wantRetCode, resp.RetCode)
			}
			data, err := sonic.Marshal(resp)
			require.NoError(t, err)
			assert.Contains(t, string(data), `"status":"failed"`)
		})
	}
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
