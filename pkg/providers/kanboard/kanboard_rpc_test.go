package kanboard

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

type jsonRPCRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     any             `json:"id"`
}

func newKanboardRPCServer(t *testing.T, handlers map[string]func(params json.RawMessage) any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var req jsonRPCRequest
		assert.NoError(t, sonic.Unmarshal(body, &req))

		handler, ok := handlers[req.Method]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32601,"message":"method not found"},"id":null}`))
			return
		}

		result := handler(req.Params)
		resp := map[string]any{
			"jsonrpc": "2.0",
			"result":  result,
			"id":      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, sonic.ConfigDefault.NewEncoder(w).Encode(resp))
	}))
}

func TestSetAuthHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		password string
		want     string
	}{
		{name: "basic credentials encoded", username: "admin", password: "secret", want: "Basic YWRtaW46c2VjcmV0"},
		{name: "empty password", username: "user", password: "", want: "Basic dXNlcjo="},
		{name: "empty username", username: "", password: "pass", want: "Basic OnBhc3M="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			header := http.Header{}
			setAuthHeader(header, tt.username, tt.password)
			assert.Equal(t, tt.want, header.Get("Authorization"))
		})
	}
}

func TestAuthTransport_RoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "sets authorization header on request", username: "kb", password: "token"},
		{name: "works with empty credentials", username: "", password: ""},
		{name: "works with special characters", username: "user@host", password: "p@ss"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			transport := &AuthTransport{
				Transport: http.DefaultTransport,
				Username:  tt.username,
				Password:  tt.password,
			}
			req, err := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.True(t, strings.HasPrefix(gotAuth, "Basic "))
		})
	}
}

func TestGetWebhookToken(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		want    string
	}{
		{name: "missing config returns empty", configs: json.RawMessage(`{}`), want: ""},
		{name: "reads token", configs: json.RawMessage(`{"kanboard":{"webhook_token":"abc123"}}`), want: "abc123"},
		{name: "empty token value", configs: json.RawMessage(`{"kanboard":{"webhook_token":""}}`), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			assert.Equal(t, tt.want, GetWebhookToken())
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantErr bool
	}{
		{name: "missing endpoint returns error", configs: json.RawMessage(`{}`), wantErr: true},
		{name: "configured endpoint returns client", configs: json.RawMessage(`{"kanboard":{"endpoint":"http://localhost/jsonrpc.php","username":"u","password":"p"}}`), wantErr: false},
		{name: "endpoint only without credentials", configs: json.RawMessage(`{"kanboard":{"endpoint":"http://localhost/jsonrpc.php"}}`), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c, err := GetClient()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, c)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestKanboard_CreateTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		result  int64
		rpcErr  bool
		wantErr bool
		wantID  int64
	}{
		{name: "creates task", result: 99, wantID: 99},
		{name: "zero id returned", result: 0, wantID: 0},
		{name: "rpc error", rpcErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
				"createTask": func(_ json.RawMessage) any {
					if tt.rpcErr {
						return nil
					}
					return tt.result
				},
			})
			if tt.rpcErr {
				srv.Close()
				srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"fail"},"id":1}`))
				}))
			}
			defer srv.Close()

			kb, err := NewKanboard(srv.URL, "admin", "secret")
			require.NoError(t, err)

			id, err := kb.CreateTask(context.Background(), &Task{Title: "Test"})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestKanboard_GetMe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{name: "returns authenticated user", user: &User{ID: 1, Username: "admin", Role: "app-admin"}},
		{name: "empty user", user: &User{}},
		{name: "rpc failure", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var srv *httptest.Server
			if tt.wantErr {
				srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"unauthorized"},"id":1}`))
				}))
			} else {
				srv = newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
					"getMe": func(_ json.RawMessage) any { return tt.user },
				})
			}
			defer srv.Close()

			kb, err := NewKanboard(srv.URL, "u", "p")
			require.NoError(t, err)

			user, err := kb.GetMe(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.user.Username, user.Username)
			assert.Equal(t, tt.user.ID, user.ID)
		})
	}
}

func TestKanboard_GetTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
	}{
		{name: "returns task", task: &Task{ID: 1, Title: "Hello"}},
		{name: "empty task", task: &Task{}},
		{name: "rpc failure", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var srv *httptest.Server
			if tt.wantErr {
				srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"not found"},"id":1}`))
				}))
			} else {
				srv = newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
					"getTask": func(_ json.RawMessage) any { return tt.task },
				})
			}
			defer srv.Close()

			kb, err := NewKanboard(srv.URL, "u", "p")
			require.NoError(t, err)

			task, err := kb.GetTask(context.Background(), 1)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.task.Title, task.Title)
		})
	}
}

func TestKanboard_GetAllTasks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tasks   []*Task
		wantLen int
		wantErr bool
	}{
		{name: "returns tasks", tasks: []*Task{{ID: 1}, {ID: 2}}, wantLen: 2},
		{name: "empty list", tasks: []*Task{}, wantLen: 0},
		{name: "rpc error", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var srv *httptest.Server
			if tt.wantErr {
				srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"fail"},"id":1}`))
				}))
			} else {
				srv = newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
					"getAllTasks": func(_ json.RawMessage) any { return tt.tasks },
				})
			}
			defer srv.Close()

			kb, err := NewKanboard(srv.URL, "u", "p")
			require.NoError(t, err)

			tasks, err := kb.GetAllTasks(context.Background(), 1, Active)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, tasks, tt.wantLen)
		})
	}
}

func TestKanboard_UpdateAndCloseTask(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		call   func(context.Context, *Kanboard) (bool, error)
	}{
		{
			name:   "update task",
			method: "updateTask",
			call: func(ctx context.Context, kb *Kanboard) (bool, error) {
				return kb.UpdateTask(ctx, 1, &Task{Title: "Updated"})
			},
		},
		{
			name:   "close task",
			method: "closeTask",
			call: func(ctx context.Context, kb *Kanboard) (bool, error) {
				return kb.CloseTask(ctx, 1)
			},
		},
		{
			name:   "open task",
			method: "openTask",
			call: func(ctx context.Context, kb *Kanboard) (bool, error) {
				return kb.OpenTask(ctx, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
				tt.method: func(_ json.RawMessage) any { return true },
			})
			defer srv.Close()

			kb, err := NewKanboard(srv.URL, "u", "p")
			require.NoError(t, err)

			ok, err := tt.call(context.Background(), kb)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}

func TestKanboard_GetColumnsAndSearch(t *testing.T) {
	t.Parallel()
	srv := newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
		"getColumns": func(_ json.RawMessage) any {
			return []types.KV{{"id": 1, "title": "Backlog"}}
		},
		"searchTasks": func(_ json.RawMessage) any {
			return []*Task{{ID: 5, Title: "Match"}}
		},
	})
	defer srv.Close()

	kb, err := NewKanboard(srv.URL, "u", "p")
	require.NoError(t, err)

	cols, err := kb.GetColumns(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, cols, 1)

	tasks, err := kb.SearchTasks(context.Background(), 1, "Match")
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "Match", tasks[0].Title)
}

func TestKanboard_RemainingMethods(t *testing.T) {
	t.Parallel()
	srv := newKanboardRPCServer(t, map[string]func(json.RawMessage) any{
		"removeTask":            func(_ json.RawMessage) any { return true },
		"moveTaskPosition":      func(_ json.RawMessage) any { return true },
		"getTaskMetadata":       func(_ json.RawMessage) any { return []TaskMetadata{{"k": "v"}} },
		"getTaskMetadataByName": func(_ json.RawMessage) any { return "meta-value" },
		"saveTaskMetadata":      func(_ json.RawMessage) any { return true },
		"removeTaskMetadata":    func(_ json.RawMessage) any { return true },
		"getAllTags":            func(_ json.RawMessage) any { return []Tag{{ID: "1", Name: "go"}} },
		"getTagsByProject":      func(_ json.RawMessage) any { return []Tag{{ID: "2", Name: "dev"}} },
		"createTag":             func(_ json.RawMessage) any { return int64(3) },
		"updateTag":             func(_ json.RawMessage) any { return true },
		"removeTag":             func(_ json.RawMessage) any { return true },
		"setTaskTags":           func(_ json.RawMessage) any { return true },
		"getTaskTags":           func(_ json.RawMessage) any { return map[string]string{"go": "go"} },
		"createSubtask":         func(_ json.RawMessage) any { return int64(10) },
		"getSubtask":            func(_ json.RawMessage) any { return &Subtask{ID: "10", Title: "sub"} },
		"getAllSubtasks":        func(_ json.RawMessage) any { return []*Subtask{{ID: "10"}} },
		"updateSubtask":         func(_ json.RawMessage) any { return true },
		"removeSubtask":         func(_ json.RawMessage) any { return true },
		"hasSubtaskTimer":       func(_ json.RawMessage) any { return true },
		"setSubtaskStartTime":   func(_ json.RawMessage) any { return true },
		"setSubtaskEndTime":     func(_ json.RawMessage) any { return true },
		"getSubtaskTimeSpent":   func(_ json.RawMessage) any { return 1.5 },
	})
	defer srv.Close()

	kb, err := NewKanboard(srv.URL, "u", "p")
	require.NoError(t, err)
	ctx := context.Background()

	ok, err := kb.RemoveTask(ctx, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.MoveTaskPosition(ctx, 1, 2, 3, 1, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	meta, err := kb.GetTaskMetadata(ctx, 1)
	require.NoError(t, err)
	require.Len(t, meta, 1)

	value, err := kb.GetTaskMetadataByName(ctx, 1, "k")
	require.NoError(t, err)
	assert.Equal(t, "meta-value", value)

	ok, err = kb.SaveTaskMetadata(ctx, 1, TaskMetadata{"k": "v"})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.RemoveTaskMetadata(ctx, 1, "k")
	require.NoError(t, err)
	assert.True(t, ok)

	tags, err := kb.GetAllTags(ctx)
	require.NoError(t, err)
	require.Len(t, tags, 1)

	tags, err = kb.GetTagsByProject(ctx, 1)
	require.NoError(t, err)
	require.Len(t, tags, 1)

	tagID, err := kb.CreateTag(ctx, 1, "new", "1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), tagID)

	ok, err = kb.UpdateTag(ctx, 1, "updated", "2")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.RemoveTag(ctx, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.SetTaskTags(ctx, 1, 2, []string{"go"})
	require.NoError(t, err)
	assert.True(t, ok)

	taskTags, err := kb.GetTaskTags(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, "go", taskTags["go"])

	subID, err := kb.CreateSubtask(ctx, 1, "sub", 1, 60, 0, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(10), subID)

	sub, err := kb.GetSubtask(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, "sub", sub.Title)

	subs, err := kb.GetAllSubtasks(ctx, 1)
	require.NoError(t, err)
	require.Len(t, subs, 1)

	ok, err = kb.UpdateSubtask(ctx, 10, 1, "updated", 1, 60, 30, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.RemoveSubtask(ctx, 10)
	require.NoError(t, err)
	assert.True(t, ok)

	hasTimer, err := kb.HasSubtaskTimer(ctx, 10, 1)
	require.NoError(t, err)
	assert.True(t, hasTimer)

	ok, err = kb.SetSubtaskStartTime(ctx, 10, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = kb.SetSubtaskEndTime(ctx, 10, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	spent, err := kb.GetSubtaskTimeSpent(ctx, 10, 1)
	require.NoError(t, err)
	assert.InDelta(t, 1.5, spent, 0.001)
}
