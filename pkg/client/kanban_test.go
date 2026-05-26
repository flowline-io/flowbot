package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestKanbanList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		projectID  int
		status     kanboard.StatusId
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:      "list active tasks",
			projectID: 1,
			status:    kanboard.Active,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"task1"},{"id":2,"title":"task2"}]}`))
			},
			wantCount: 2,
		},
		{
			name:      "list empty project",
			projectID: 1,
			status:    kanboard.Inactive,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
		{
			name:       "invalid project id zero",
			projectID:  0,
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:       "invalid project id negative",
			projectID:  -5,
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:      "api error",
			projectID: 1,
			status:    kanboard.Active,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"kanban unavailable"}`))
			},
			wantErr:    true,
			errContain: "kanban unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.List(context.Background(), tt.projectID, tt.status)

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

func TestKanbanListAll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		projectID  int
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:      "list all tasks",
			projectID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1},{"id":2},{"id":3}]}`))
			},
			wantCount: 3,
		},
		{
			name:      "empty project",
			projectID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
		{
			name:       "invalid project id",
			projectID:  0,
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name: "api error",
			projectID: 1,
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.ListAll(context.Background(), tt.projectID)

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

func TestKanbanGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		handler    http.HandlerFunc
		wantTitle  string
		wantErr    bool
		errContain string
	}{
		{
			name: "task found",
			id:   1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"title":"My Task","project_id":1}}`))
			},
			wantTitle: "My Task",
		},
		{
			name:       "invalid id zero",
			id:         0,
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name:       "invalid id negative",
			id:         -1,
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name: "task not found",
			id:   999,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"task not found"}`))
			},
			wantErr:    true,
			errContain: "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Get(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantTitle, result.Title)
		})
	}
}

func TestKanbanCreate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        KanbanCreateRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "create task success",
			req:  KanbanCreateRequest{Title: "New Task", Description: "details", ProjectID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":10}}`))
			},
		},
		{
			name:       "empty title",
			req:        KanbanCreateRequest{Title: ""},
			wantErr:    true,
			errContain: "title is required",
		},
		{
			name: "description at max length",
			req:  KanbanCreateRequest{Title: "Task", Description: makeLen(validate.DescMaxLen)},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":11}}`))
			},
		},
		{
			name: "api error",
			req:  KanbanCreateRequest{Title: "Task"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid data"}`))
			},
			wantErr:    true,
			errContain: "invalid data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Create(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		req        KanbanUpdateRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "update task success",
			id:   1,
			req:  KanbanUpdateRequest{Title: "Updated", Description: "new desc"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid id zero",
			id:         0,
			req:        KanbanUpdateRequest{Title: "Test"},
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name:       "invalid id negative",
			id:         -1,
			req:        KanbanUpdateRequest{Title: "Test"},
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name: "task not found",
			id:   999,
			req:  KanbanUpdateRequest{Title: "Test"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"task not found"}`))
			},
			wantErr:    true,
			errContain: "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Update(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.Success)
		})
	}
}

func TestKanbanClose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "close task success",
			id:   1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid id zero",
			id:         0,
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name: "task already closed",
			id:   1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"task not found"}`))
			},
			wantErr:    true,
			errContain: "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Close(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		req        KanbanMoveRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "move task success",
			id:   1,
			req:  KanbanMoveRequest{ColumnID: 2, Position: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid id",
			id:         0,
			req:        KanbanMoveRequest{ColumnID: 1},
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name:       "invalid column id",
			id:         1,
			req:        KanbanMoveRequest{ColumnID: 0},
			wantErr:    true,
			errContain: "column_id must be positive",
		},
		{
			name:       "negative position",
			id:         1,
			req:        KanbanMoveRequest{ColumnID: 1, Position: -1},
			wantErr:    true,
			errContain: "position must be non-negative",
		},
		{
			name: "api error",
			id:   1,
			req:  KanbanMoveRequest{ColumnID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid move"}`))
			},
			wantErr:    true,
			errContain: "invalid move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Move(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanListColumns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		projectID  int
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:      "list columns",
			projectID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"Backlog"},{"id":2,"title":"In Progress"},{"id":3,"title":"Done"}]}`))
			},
			wantCount: 3,
		},
		{
			name:       "invalid project id",
			projectID:  0,
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:      "api error",
			projectID: 1,
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.ListColumns(context.Background(), tt.projectID)

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

func TestKanbanSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		projectID  int
		query      string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:      "search finds results",
			projectID: 1,
			query:     "bug",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":5,"title":"fix bug in login"}]}`))
			},
			wantCount: 1,
		},
		{
			name:       "invalid project id",
			projectID:  0,
			query:      "test",
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:       "empty query",
			projectID:  1,
			query:      "",
			wantErr:    true,
			errContain: "query is required",
		},
		{
			name:      "search no results",
			projectID: 1,
			query:     "no-match",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.Search(context.Background(), tt.projectID, tt.query)

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

func TestKanbanGetMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:   "metadata found",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"name":"type","value":"bug"},{"name":"priority","value":"high"}]}`))
			},
			wantCount: 2,
		},
		{
			name:       "invalid task id",
			taskID:     0,
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:   "no metadata",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.GetMetadata(context.Background(), tt.taskID)

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

func TestKanbanGetMetadataByName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		metaName   string
		handler    http.HandlerFunc
		want       string
		wantErr    bool
		errContain string
	}{
		{
			name:     "retrieve metadata value",
			taskID:   1,
			metaName: "priority",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":"high"}`))
			},
			want: "high",
		},
		{
			name:       "invalid task id",
			taskID:     0,
			metaName:   "priority",
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:       "empty name",
			taskID:     1,
			metaName:   "",
			wantErr:    true,
			errContain: "name is required",
		},
		{
			name:     "metadata not found",
			taskID:   1,
			metaName: "nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"not found"}`))
			},
			wantErr:    true,
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.GetMetadataByName(context.Background(), tt.taskID, tt.metaName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestKanbanSaveMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:   "save metadata success",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid task id",
			taskID:     0,
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:   "api error",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid metadata"}`))
			},
			wantErr:    true,
			errContain: "invalid metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.SaveMetadata(context.Background(), tt.taskID, kanboard.TaskMetadata{"type": "bug"})

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanRemoveMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		metaName   string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:     "remove metadata success",
			taskID:   1,
			metaName: "type",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid task id",
			taskID:     0,
			metaName:   "type",
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:       "empty name",
			taskID:     1,
			metaName:   "",
			wantErr:    true,
			errContain: "name is required",
		},
		{
			name:     "api error",
			taskID:   1,
			metaName: "type",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"metadata not found"}`))
			},
			wantErr:    true,
			errContain: "metadata not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.RemoveMetadata(context.Background(), tt.taskID, tt.metaName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanTags(t *testing.T) {
	t.Parallel()
	t.Run("list tags", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":"t1","name":"bug","project_id":"1"}]}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.ListTags(context.Background())
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "bug", result[0].Name)
	})

	t.Run("list tags empty", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.ListTags(context.Background())
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("list tags api error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		_, err := c.Kanban.ListTags(context.Background())
		require.Error(t, err)
	})
}

func TestKanbanListTagsByProject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		projectID  int
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:      "list tags by project",
			projectID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":"t1","name":"bug"},{"id":"t2","name":"feature"}]}`))
			},
			wantCount: 2,
		},
		{
			name:       "invalid project id",
			projectID:  0,
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:      "api error",
			projectID: 1,
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.ListTagsByProject(context.Background(), tt.projectID)

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

func TestKanbanCreateTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        KanbanCreateTagRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "create tag success",
			req:  KanbanCreateTagRequest{ProjectID: 1, Name: "bug"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":5}}`))
			},
		},
		{
			name:       "invalid project id",
			req:        KanbanCreateTagRequest{ProjectID: 0, Name: "tag"},
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:       "empty name",
			req:        KanbanCreateTagRequest{ProjectID: 1, Name: ""},
			wantErr:    true,
			errContain: "name is required",
		},
		{
			name: "api error",
			req:  KanbanCreateTagRequest{ProjectID: 1, Name: "duplicate"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"status":"failed","message":"tag already exists"}`))
			},
			wantErr:    true,
			errContain: "tag already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.CreateTag(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanUpdateTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		req        KanbanUpdateTagRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "update tag success",
			id:   1,
			req:  KanbanUpdateTagRequest{Name: "updated-name"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid id",
			id:         0,
			req:        KanbanUpdateTagRequest{Name: "name"},
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name:       "empty name",
			id:         1,
			req:        KanbanUpdateTagRequest{Name: ""},
			wantErr:    true,
			errContain: "name is required",
		},
		{
			name: "tag not found",
			id:   999,
			req:  KanbanUpdateTagRequest{Name: "name"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"tag not found"}`))
			},
			wantErr:    true,
			errContain: "tag not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.UpdateTag(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanRemoveTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "remove tag success",
			id:   1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid id",
			id:         0,
			wantErr:    true,
			errContain: "id must be positive",
		},
		{
			name: "tag not found",
			id:   999,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"tag not found"}`))
			},
			wantErr:    true,
			errContain: "tag not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.RemoveTag(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanGetTaskTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:   "get task tags",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"t1":"bug","t2":"feature"}}`))
			},
			wantCount: 2,
		},
		{
			name:       "invalid task id",
			taskID:     0,
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:   "no tags",
			taskID: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{}}`))
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.GetTaskTags(context.Background(), tt.taskID)

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

func TestKanbanSetTaskTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		taskID     int
		req        KanbanSetTaskTagsRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:   "set task tags success",
			taskID: 1,
			req:    KanbanSetTaskTagsRequest{ProjectID: 1, Tags: []string{"tag1", "tag2"}},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "invalid task id",
			taskID:     0,
			req:        KanbanSetTaskTagsRequest{ProjectID: 1},
			wantErr:    true,
			errContain: "task_id must be positive",
		},
		{
			name:       "invalid project id",
			taskID:     1,
			req:        KanbanSetTaskTagsRequest{ProjectID: 0},
			wantErr:    true,
			errContain: "project_id must be positive",
		},
		{
			name:   "api error",
			taskID: 1,
			req:    KanbanSetTaskTagsRequest{ProjectID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid tags"}`))
			},
			wantErr:    true,
			errContain: "invalid tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Kanban.SetTaskTags(context.Background(), tt.taskID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestKanbanSubtasks(t *testing.T) {
	t.Parallel()
	t.Run("list subtasks", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":"1","title":"sub1"},{"id":"2","title":"sub2"}]}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.ListSubtasks(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("list subtasks invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.ListSubtasks(context.Background(), 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("get subtask", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"1","title":"sub1","task_id":"1"}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.GetSubtask(context.Background(), 1, 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "sub1", result.Title)
	})

	t.Run("get subtask invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.GetSubtask(context.Background(), 0, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("get subtask invalid subtask id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.GetSubtask(context.Background(), 1, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtask_id must be positive")
	})

	t.Run("create subtask", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"id":5}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.CreateSubtask(context.Background(), 1, KanbanCreateSubtaskRequest{Title: "new sub"})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(5), result.ID)
	})

	t.Run("create subtask invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.CreateSubtask(context.Background(), 0, KanbanCreateSubtaskRequest{Title: "sub"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("create subtask empty title", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.CreateSubtask(context.Background(), 1, KanbanCreateSubtaskRequest{Title: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("update subtask", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.UpdateSubtask(context.Background(), 1, 1, KanbanUpdateSubtaskRequest{Title: "updated"})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("update subtask invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.UpdateSubtask(context.Background(), 0, 1, KanbanUpdateSubtaskRequest{Title: "up"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("update subtask invalid subtask id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.UpdateSubtask(context.Background(), 1, 0, KanbanUpdateSubtaskRequest{Title: "up"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtask_id must be positive")
	})

	t.Run("remove subtask", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.RemoveSubtask(context.Background(), 1, 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("remove subtask invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.RemoveSubtask(context.Background(), 0, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("remove subtask invalid subtask id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.RemoveSubtask(context.Background(), 1, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtask_id must be positive")
	})
}

func TestKanbanSubtaskTimer(t *testing.T) {
	t.Parallel()
	t.Run("has subtask timer", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"result":true}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.HasSubtaskTimer(context.Background(), 1, 1, 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Result)
	})

	t.Run("has subtask timer invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.HasSubtaskTimer(context.Background(), 0, 1, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("has subtask timer invalid subtask id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.HasSubtaskTimer(context.Background(), 1, 0, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtask_id must be positive")
	})

	t.Run("set subtask start time", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"result":true}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.SetSubtaskStartTime(context.Background(), 1, 1, 42)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("set subtask start time invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.SetSubtaskStartTime(context.Background(), 0, 1, 42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})

	t.Run("set subtask end time", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"result":true}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.SetSubtaskEndTime(context.Background(), 1, 1, 42)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("set subtask end time invalid subtask id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.SetSubtaskEndTime(context.Background(), 1, 0, 42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subtask_id must be positive")
	})

	t.Run("get subtask time spent", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"result":2.5}}`))
		}))
		defer server.Close()

		c := NewClient(server.URL, "token")
		result, err := c.Kanban.GetSubtaskTimeSpent(context.Background(), 1, 1, 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.InEpsilon(t, 2.5, result.Result, 0)
	})

	t.Run("get subtask time spent invalid task id", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		_, err := c.Kanban.GetSubtaskTimeSpent(context.Background(), 0, 1, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_id must be positive")
	})
}

func TestValidateCreateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		req        *KanbanCreateRequest
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid request",
			req:     &KanbanCreateRequest{Title: "My Task", Description: "details"},
			wantErr: false,
		},
		{
			name:       "empty title",
			req:        &KanbanCreateRequest{Title: ""},
			wantErr:    true,
			errContain: "title is required",
		},
		{
			name:    "title at max length",
			req:     &KanbanCreateRequest{Title: makeLen(validate.TitleMaxLen)},
			wantErr: false,
		},
		{
			name:       "title exceeds max length",
			req:        &KanbanCreateRequest{Title: makeLen(validate.TitleMaxLen + 1)},
			wantErr:    true,
			errContain: "title exceeds maximum length",
		},
		{
			name:       "description exceeds max length",
			req:        &KanbanCreateRequest{Title: "Task", Description: makeLen(validate.DescMaxLen + 1)},
			wantErr:    true,
			errContain: "description exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCreateRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		req        *KanbanUpdateRequest
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid update with title",
			req:     &KanbanUpdateRequest{Title: "Updated", Description: "new desc"},
			wantErr: false,
		},
		{
			name:    "valid update description only",
			req:     &KanbanUpdateRequest{Description: "only desc"},
			wantErr: false,
		},
		{
			name:    "valid update empty request",
			req:     &KanbanUpdateRequest{},
			wantErr: false,
		},
		{
			name:       "title exceeds max length",
			req:        &KanbanUpdateRequest{Title: makeLen(validate.TitleMaxLen + 1)},
			wantErr:    true,
			errContain: "title exceeds maximum length",
		},
		{
			name:       "description exceeds max length",
			req:        &KanbanUpdateRequest{Description: makeLen(validate.DescMaxLen + 1)},
			wantErr:    true,
			errContain: "description exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateUpdateRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateMoveRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		req        *KanbanMoveRequest
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid move request",
			req:     &KanbanMoveRequest{ColumnID: 2, Position: 1},
			wantErr: false,
		},
		{
			name:       "invalid column id zero",
			req:        &KanbanMoveRequest{ColumnID: 0},
			wantErr:    true,
			errContain: "column_id must be positive",
		},
		{
			name:       "negative position",
			req:        &KanbanMoveRequest{ColumnID: 1, Position: -1},
			wantErr:    true,
			errContain: "position must be non-negative",
		},
		{
			name:       "negative swimlane id",
			req:        &KanbanMoveRequest{ColumnID: 1, SwimlaneID: -1},
			wantErr:    true,
			errContain: "swimlane_id must be non-negative",
		},
		{
			name:       "negative project id",
			req:        &KanbanMoveRequest{ColumnID: 1, ProjectID: -1},
			wantErr:    true,
			errContain: "project_id must be non-negative",
		},
		{
			name:    "position zero is valid",
			req:     &KanbanMoveRequest{ColumnID: 1, Position: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateMoveRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func makeLen(n int) string {
	result := make([]byte, n)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}


