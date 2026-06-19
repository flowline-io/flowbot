package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTestScheduleStore(t *testing.T, fn func()) {
	t.Helper()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()
	fn()
}

func TestScheduleTaskToolValidation(t *testing.T) {
	t.Parallel()
	tool := ScheduleTaskTool{deps: ScheduleToolDeps{UID: types.Uid("user:test"), SessionID: "sess-1"}}

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{name: "missing name", args: map[string]any{"prompt": "do work"}, wantErr: true},
		{name: "missing prompt", args: map[string]any{"name": "daily"}, wantErr: true},
		{name: "missing schedule", args: map[string]any{"name": "daily", "prompt": "check logs"}, wantErr: true},
		{name: "both cron and run_at", args: map[string]any{"name": "x", "prompt": "p", "cron": "0 9 * * *", "run_at": "2026-06-20T09:00:00Z"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "call-1", tt.args, nil)
			require.NoError(t, err)
			if tt.wantErr {
				assert.True(t, result.IsError)
				return
			}
			assert.False(t, result.IsError)
		})
	}
}

func TestScheduleTaskToolCreateCron(t *testing.T) {
	withTestScheduleStore(t, func() {
		sched := NewTaskScheduler()
		SetGlobalScheduler(sched)
		require.NoError(t, sched.Start(context.Background()))
		defer func() { _ = sched.Stop(context.Background()) }()

		tool := ScheduleTaskTool{deps: ScheduleToolDeps{UID: types.Uid("user:alice"), SessionID: "sess-src"}}
		result, err := tool.Execute(context.Background(), "call-1", map[string]any{
			"name":   "daily check",
			"prompt": "Summarize workspace status",
			"cron":   "0 9 * * *",
		}, nil)
		require.NoError(t, err)
		require.False(t, result.IsError)
		assert.Contains(t, scheduledToolResultText(result), "created")

		tasks, err := store.Database.ListChatScheduledTasks(context.Background(), store.ListChatScheduledTasksOptions{
			UID: "user:alice",
		})
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		assert.Equal(t, string(schema.ChatScheduledTaskKindCron), tasks[0].ScheduleKind)
	})
}

func TestUpdateScheduledTaskToolRejectsKindMismatch(t *testing.T) {
	withTestScheduleStore(t, func() {
		runAt := time.Now().UTC().Add(2 * time.Hour)
		require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
			Flag:         "task-once",
			UID:          "user:alice",
			Name:         "reminder",
			ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
			Prompt:       "remind me",
			RunAt:        &runAt,
			State:        string(schema.ChatScheduledTaskStateActive),
		}))

		tool := UpdateScheduledTaskTool{deps: ScheduleToolDeps{UID: types.Uid("user:alice")}}
		result, err := tool.Execute(context.Background(), "call-2", map[string]any{
			"task_id": "task-once",
			"cron":    "0 8 * * *",
		}, nil)
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestCancelScheduledTaskTool(t *testing.T) {
	withTestScheduleStore(t, func() {
		require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
			Flag:         "task-1",
			UID:          "user:alice",
			Name:         "job",
			ScheduleKind: string(schema.ChatScheduledTaskKindCron),
			Cron:         "0 9 * * *",
			Prompt:       "work",
			State:        string(schema.ChatScheduledTaskStateActive),
		}))

		tool := CancelScheduledTaskTool{deps: ScheduleToolDeps{UID: types.Uid("user:alice")}}
		result, err := tool.Execute(context.Background(), "call-3", map[string]any{"task_id": "task-1"}, nil)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		task, err := store.Database.GetChatScheduledTask(context.Background(), "task-1")
		require.NoError(t, err)
		assert.Equal(t, string(schema.ChatScheduledTaskStateCancelled), task.State)
	})
}

func TestUpdateScheduledTaskToolPause(t *testing.T) {
	withTestScheduleStore(t, func() {
		sched := NewTaskScheduler()
		SetGlobalScheduler(sched)
		require.NoError(t, sched.Start(context.Background()))
		defer func() { _ = sched.Stop(context.Background()) }()

		require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
			Flag:         "task-pause",
			UID:          "user:alice",
			Name:         "job",
			ScheduleKind: string(schema.ChatScheduledTaskKindCron),
			Cron:         "0 9 * * *",
			Prompt:       "work",
			State:        string(schema.ChatScheduledTaskStateActive),
		}))

		tool := UpdateScheduledTaskTool{deps: ScheduleToolDeps{UID: types.Uid("user:alice")}}
		result, err := tool.Execute(context.Background(), "call-4", map[string]any{
			"task_id": "task-pause",
			"state":   string(schema.ChatScheduledTaskStatePaused),
		}, nil)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		task, err := store.Database.GetChatScheduledTask(context.Background(), "task-pause")
		require.NoError(t, err)
		assert.Equal(t, string(schema.ChatScheduledTaskStatePaused), task.State)
	})
}

func scheduledToolResultText(result msg.ToolResultMessage) string {
	if len(result.Parts) == 0 {
		return ""
	}
	if part, ok := result.Parts[0].(msg.TextPart); ok {
		return part.Text
	}
	return ""
}
