package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "lists pipelines",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":{
						"pipelines":[
							{"name":"p1","description":"desc1","enabled":true,"trigger":{"event":"e1"},"steps":[]},
							{"name":"p2","description":"desc2","enabled":false,"trigger":{"event":"e2"},"steps":[]}
						]
					}
				}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty pipeline list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"pipelines":[]}}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "api error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"pipeline service down"}`))
			},
			wantErr:    true,
			errContain: "pipeline service down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Pipeline.List(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Pipelines, tt.wantCount)
		})
	}
}

func TestPipelineRun(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantMsg    string
		wantErr    bool
		errContain string
	}{
		{
			name: "runs pipeline successfully",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"message":"pipeline started"}}`))
			},
			wantMsg: "pipeline started",
			wantErr: false,
		},
		{
			name: "pipeline not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"pipeline not found"}`))
			},
			wantErr:    true,
			errContain: "pipeline not found",
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
			result, err := c.Pipeline.Run(context.Background(), "my-pipeline")

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
