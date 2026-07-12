package web

import (
	"cmp"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func (s *testStore) ListChatScheduledTasks(_ context.Context, opts store.ListChatScheduledTasksOptions) ([]*gen.ChatScheduledTask, error) {
	if s.chatScheduledTasksErr != nil {
		return nil, s.chatScheduledTasksErr
	}
	rows := append([]*gen.ChatScheduledTask(nil), s.chatScheduledTasks...)
	filtered := rows[:0]
	for _, task := range rows {
		if opts.UID != "" && task.UID != opts.UID {
			continue
		}
		if len(opts.States) > 0 && !slices.Contains(opts.States, task.State) {
			continue
		}
		filtered = append(filtered, task)
	}
	rows = filtered
	slices.SortFunc(rows, func(a, b *gen.ChatScheduledTask) int {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return rows, nil
}

func (s *testStore) GetChatScheduledTaskForUID(_ context.Context, flag, uid string) (*gen.ChatScheduledTask, error) {
	if s.chatScheduledTasksByFlag != nil {
		task, ok := s.chatScheduledTasksByFlag[flag]
		if !ok || task.UID != uid {
			return nil, types.ErrNotFound
		}
		return task, nil
	}
	for _, task := range s.chatScheduledTasks {
		if task.Flag == flag && task.UID == uid {
			return task, nil
		}
	}
	return nil, types.ErrNotFound
}

func (s *testStore) GetChatScheduledTask(_ context.Context, flag string) (*gen.ChatScheduledTask, error) {
	if s.chatScheduledTasksByFlag != nil {
		task, ok := s.chatScheduledTasksByFlag[flag]
		if !ok {
			return nil, types.ErrNotFound
		}
		return task, nil
	}
	for _, task := range s.chatScheduledTasks {
		if task.Flag == flag {
			return task, nil
		}
	}
	return nil, types.ErrNotFound
}

func (s *testStore) UpdateChatScheduledTask(_ context.Context, flag string, params store.UpdateChatScheduledTaskParams) error {
	task, err := s.GetChatScheduledTask(context.Background(), flag)
	if err != nil {
		return err
	}
	if params.State != nil {
		task.State = *params.State
	}
	task.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *testStore) ListChatScheduledTaskRuns(_ context.Context, taskID string, limit int) ([]*gen.ChatScheduledTaskRun, error) {
	if s.chatScheduledTaskRunsErr != nil {
		return nil, s.chatScheduledTaskRunsErr
	}
	rows := append([]*gen.ChatScheduledTaskRun(nil), s.chatScheduledTaskRuns[taskID]...)
	slices.SortFunc(rows, func(a, b *gen.ChatScheduledTaskRun) int {
		if c := b.StartedAt.Compare(a.StartedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func TestListScheduledTaskModels(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name     string
		store    *testStore
		wantLen  int
		wantName string
		wantErr  error
	}{
		{
			name: "returns all lifecycle states for current user",
			store: &testStore{
				chatScheduledTasks: []*gen.ChatScheduledTask{
					{ID: 1, Flag: "task-active", UID: "testuser", Name: "Active Task", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now},
					{ID: 2, Flag: "task-failed", UID: "testuser", Name: "Failed Task", State: string(schema.ChatScheduledTaskStateFailed), UpdatedAt: now.Add(time.Minute), CreatedAt: now},
					{ID: 3, Flag: "task-other", UID: "someone-else", Name: "Other Task", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now.Add(2 * time.Minute), CreatedAt: now},
				},
			},
			wantLen:  2,
			wantName: "Failed Task",
		},
		{
			name:    "propagates store error",
			store:   &testStore{chatScheduledTasksErr: errors.New("boom")},
			wantErr: errors.New("boom"),
		},
		{
			name:    "unauthorized without request context",
			store:   &testStore{},
			wantErr: types.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				app := setupAuthenticatedApp(t, tt.store)
				req := httptest.NewRequest(http.MethodGet, "/service/web/agent-scheduled-tasks/list", http.NoBody)
				if tt.wantErr != types.ErrUnauthorized {
					req.Header.Set("Cookie", "accessToken=test-token")
				}
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)

				if tt.wantErr != nil {
					assert.NotEqual(t, http.StatusOK, resp.StatusCode)
					return
				}
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Contains(t, string(body), tt.wantName)
				if tt.wantLen == 0 {
					assert.Contains(t, string(body), "No scheduled tasks found")
				}
			})
		})
	}
}

func TestAgentScheduledTasksPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "page redirects to login", path: "/service/web/agent-scheduled-tasks"},
		{name: "list redirects to login", path: "/service/web/agent-scheduled-tasks/list"},
		{name: "detail redirects to login", path: "/service/web/agent-scheduled-tasks/task-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		})
	}
}

func TestAgentScheduledTasksListAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name     string
		path     string
		tasks    []*gen.ChatScheduledTask
		wantBody string
	}{
		{
			name: "list page contains table",
			path: "/service/web/agent-scheduled-tasks",
			tasks: []*gen.ChatScheduledTask{
				{ID: 1, Flag: "task-demo", UID: "testuser", Name: "Nightly Summary", ScheduleKind: "cron", Cron: "0 0 * * *", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: `data-testid="agent-scheduled-tasks-table"`,
		},
		{
			name: "table partial renders scheduled task row",
			path: "/service/web/agent-scheduled-tasks/list",
			tasks: []*gen.ChatScheduledTask{
				{ID: 1, Flag: "task-table", UID: "testuser", Name: "Weekly Digest", ScheduleKind: "cron", Cron: "0 9 * * 1", State: string(schema.ChatScheduledTaskStatePaused), UpdatedAt: now, CreatedAt: now},
			},
			wantBody: "Weekly Digest",
		},
		{
			name:     "empty list shows placeholder",
			path:     "/service/web/agent-scheduled-tasks/list",
			tasks:    nil,
			wantBody: "No scheduled tasks found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{chatScheduledTasks: tt.tasks}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
				req.Header.Set("Cookie", "accessToken=test-token")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentScheduledTaskDetailAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	finishedAt := now.Add(2 * time.Minute)
	tests := []struct {
		name       string
		path       string
		tasks      map[string]*gen.ChatScheduledTask
		runs       map[string][]*gen.ChatScheduledTaskRun
		wantStatus int
		wantBody   string
	}{
		{
			name: "detail renders task prompt and runs",
			path: "/service/web/agent-scheduled-tasks/task-detail",
			tasks: map[string]*gen.ChatScheduledTask{
				"task-detail": {
					ID: 1, Flag: "task-detail", UID: "testuser", Name: "Daily Check", Prompt: "check system health",
					ScheduleKind: "cron", Cron: "0 * * * *", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now,
				},
			},
			runs: map[string][]*gen.ChatScheduledTaskRun{
				"task-detail": {
					{ID: 1, Flag: "run-1", TaskID: "task-detail", RunSessionID: "sess-1", State: string(schema.ChatScheduledTaskRunStateCompleted), Reply: "all green", StartedAt: now, FinishedAt: &finishedAt},
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "check system health",
		},
		{
			name: "detail includes state panel",
			path: "/service/web/agent-scheduled-tasks/task-detail",
			tasks: map[string]*gen.ChatScheduledTask{
				"task-detail": {
					ID: 1, Flag: "task-detail", UID: "testuser", Name: "Daily Check", Prompt: "check system health",
					ScheduleKind: "cron", Cron: "0 * * * *", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now,
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   `data-testid="agent-scheduled-task-state-form"`,
		},
		{
			name:       "missing task returns not found",
			path:       "/service/web/agent-scheduled-tasks/missing",
			tasks:      map[string]*gen.ChatScheduledTask{},
			wantStatus: http.StatusNotFound,
			wantBody:   "scheduled task not found",
		},
		{
			name: "detail shows empty runs message",
			path: "/service/web/agent-scheduled-tasks/task-empty",
			tasks: map[string]*gen.ChatScheduledTask{
				"task-empty": {
					ID: 2, Flag: "task-empty", UID: "testuser", Name: "One Shot", Prompt: "run once",
					ScheduleKind: "once", State: string(schema.ChatScheduledTaskStateCompleted), UpdatedAt: now, CreatedAt: now,
				},
			},
			runs:       map[string][]*gen.ChatScheduledTaskRun{},
			wantStatus: http.StatusOK,
			wantBody:   "No runs found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{
					chatScheduledTasksByFlag: tt.tasks,
					chatScheduledTaskRuns:    tt.runs,
				}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
				req.Header.Set("Cookie", "accessToken=test-token")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
			})
		})
	}
}

func TestAgentScheduledTaskSetStateAuthenticated(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		path       string
		body       string
		tasks      map[string]*gen.ChatScheduledTask
		wantStatus int
		wantBody   string
	}{
		{
			name: "updates task state and returns panel",
			path: "/service/web/agent-scheduled-tasks/task-state/state",
			body: "state=paused",
			tasks: map[string]*gen.ChatScheduledTask{
				"task-state": {
					ID: 1, Flag: "task-state", UID: "testuser", Name: "Daily", Prompt: "run",
					ScheduleKind: "cron", Cron: "0 * * * *", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now,
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   `data-testid="agent-scheduled-task-state-panel"`,
		},
		{
			name: "invalid state returns bad request",
			path: "/service/web/agent-scheduled-tasks/task-state/state",
			body: "state=archived",
			tasks: map[string]*gen.ChatScheduledTask{
				"task-state": {
					ID: 1, Flag: "task-state", UID: "testuser", Name: "Daily", Prompt: "run",
					ScheduleKind: "cron", Cron: "0 * * * *", State: string(schema.ChatScheduledTaskStateActive), UpdatedAt: now, CreatedAt: now,
				},
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid state",
		},
		{
			name:       "missing task returns not found",
			path:       "/service/web/agent-scheduled-tasks/missing/state",
			body:       "state=paused",
			tasks:      map[string]*gen.ChatScheduledTask{},
			wantStatus: http.StatusNotFound,
			wantBody:   "scheduled task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withChatAgentEnabled(t, func() {
				ts := &testStore{chatScheduledTasksByFlag: tt.tasks}
				app := setupAuthenticatedApp(t, ts)

				req := httptest.NewRequest(http.MethodPut, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Cookie", "accessToken=test-token")
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantBody)
				if tt.wantStatus == http.StatusOK {
					task := tt.tasks["task-state"]
					assert.Equal(t, string(schema.ChatScheduledTaskStatePaused), task.State)
				}
			})
		})
	}
}
