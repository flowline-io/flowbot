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
							{"name":"p1","description":"desc1","enabled":true,"triggers":["event:e1"]},
							{"name":"p2","description":"desc2","enabled":false,"triggers":["cron:@daily"]}
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
		wantRunID  int64
		wantErr    bool
		errContain string
	}{
		{
			name: "runs pipeline successfully",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"run_id":42}}`))
			},
			wantRunID: 42,
			wantErr:   false,
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
			result, err := c.Pipeline.Run(context.Background(), "my-pipeline", map[string]any{"url": "https://example.com"})

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantRunID, result.RunID)
		})
	}
}

func TestPipelineApply(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"name":"demo","enabled":true,"version":3}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "token")
	result, err := c.Pipeline.Apply(context.Background(), []byte("name: demo\nenabled: true\ntriggers: []\nsteps: []\n"))
	require.NoError(t, err)
	assert.Equal(t, "demo", result.Name)
	assert.Equal(t, 3, result.Version)
}
