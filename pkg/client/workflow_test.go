package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRunFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantMsg    string
		wantErr    bool
		errContain string
	}{
		{
			name: "runs workflow file successfully",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"message":"workflow completed"}}`))
			},
			wantMsg: "workflow completed",
			wantErr: false,
		},
		{
			name: "workflow parse error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid workflow YAML"}`))
			},
			wantErr:    true,
			errContain: "invalid workflow YAML",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"execution failed"}`))
			},
			wantErr:    true,
			errContain: "execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Workflow.RunFile(context.Background(), "/path/to/workflow.yaml")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantMsg, result.Message)
		})
	}
}
