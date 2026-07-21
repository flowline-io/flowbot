package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowClient_ApplyListGetExportDeleteRunRuns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		method     string
		pathPrefix string
		handler    http.HandlerFunc
		call       func(t *testing.T, c *Client)
	}{
		{
			name:       "apply yaml",
			method:     http.MethodPost,
			pathPrefix: "/service/workflow/apply",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"name":"demo","enabled":true}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.Apply(context.Background(), []byte("name: demo\npipeline: [a]\ntasks: [{id: a, action: mapper:}]"))
				require.NoError(t, err)
				assert.Equal(t, "demo", res.Name)
				assert.Equal(t, int64(1), res.ID)
			},
		},
		{
			name:       "list workflows",
			method:     http.MethodGet,
			pathPrefix: "/service/workflow/list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"workflows":[{"id":1,"name":"demo","enabled":true}]}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.List(context.Background())
				require.NoError(t, err)
				require.Len(t, res.Workflows, 1)
				assert.Equal(t, "demo", res.Workflows[0].Name)
			},
		},
		{
			name:       "run returns run_id",
			method:     http.MethodPost,
			pathPrefix: "/service/workflow/run",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"status":"ok","data":{"run_id":99}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.Run(context.Background(), "demo", map[string]any{"url": "https://x"})
				require.NoError(t, err)
				assert.Equal(t, int64(99), res.RunID)
			},
		},
		{
			name:       "export yaml",
			method:     http.MethodGet,
			pathPrefix: "/service/workflow/export/",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"yaml":"name: demo\n"}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.Export(context.Background(), "demo")
				require.NoError(t, err)
				assert.Contains(t, res.YAML, "name: demo")
			},
		},
		{
			name:       "delete workflow",
			method:     http.MethodDelete,
			pathPrefix: "/service/workflow/delete/",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"deleted":"demo"}}`))
			},
			call: func(t *testing.T, c *Client) {
				err := c.Workflow.Delete(context.Background(), "demo")
				require.NoError(t, err)
			},
		},
		{
			name:       "list runs",
			method:     http.MethodGet,
			pathPrefix: "/service/workflow/runs/",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"runs":[{"id":7,"workflow_name":"demo","status":1,"trigger_type":"manual"}]}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.Runs(context.Background(), "demo")
				require.NoError(t, err)
				require.Len(t, res.Runs, 1)
				assert.Equal(t, int64(7), res.Runs[0].ID)
			},
		},
		{
			name:       "get workflow",
			method:     http.MethodGet,
			pathPrefix: "/service/workflow/get/",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"demo","enabled":true,"pipeline":["a"]}}`))
			},
			call: func(t *testing.T, c *Client) {
				res, err := c.Workflow.Get(context.Background(), "demo")
				require.NoError(t, err)
				assert.Equal(t, "demo", res["name"])
			},
		},
		{
			name:       "run validation error",
			method:     http.MethodPost,
			pathPrefix: "/service/workflow/run",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"input validation failed"}`))
			},
			call: func(t *testing.T, c *Client) {
				_, err := c.Workflow.Run(context.Background(), "demo", map[string]any{})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "input validation failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.method, r.Method)
				assert.Contains(t, r.URL.Path, tt.pathPrefix)
				tt.handler(w, r)
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			tt.call(t, c)
		})
	}
}
