package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDashboard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "returns dashboard data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"widgets":["a","b"]}}`))
			},
			wantErr: false,
		},
		{
			name: "empty dashboard",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			},
			wantErr: false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"unavailable"}`))
			},
			wantErr:    true,
			errContain: "unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.User.Dashboard(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestUserMetrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantBookTotal int64
		wantErr       bool
		errContain    string
	}{
		{
			name: "returns metrics",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":{
						"bot_total_stats":10,
						"bookmark_total_stats":25,
						"kanban_task_total_stats":5
					}
				}`))
			},
			wantBookTotal: 25,
			wantErr:       false,
		},
		{
			name: "zero metrics",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bot_total_stats":0}}`))
			},
			wantBookTotal: 0,
			wantErr:       false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"metrics unavailable"}`))
			},
			wantErr:    true,
			errContain: "metrics unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.User.Metrics(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantBookTotal, result.BookmarkTotalStats)
		})
	}
}

func TestUserKanbanList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "returns kanban tasks",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":[{"id":1,"title":"task1","project_id":1},{"id":2,"title":"task2","project_id":1}]
				}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty task list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.User.KanbanList(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestUserBookmarkList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "returns bookmarks",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":[
						{"id":"b1","title":"bookmark1"},
						{"id":"b2","title":"bookmark2"}
					]
				}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty bookmark list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.User.BookmarkList(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}
