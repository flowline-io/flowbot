package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskSchedulerMarksMissedOnceTask(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()

	past := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	fixedNow := time.Date(2026, 1, 1, 9, 20, 0, 0, time.UTC)

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-missed",
		UID:          "user:alice",
		Name:         "late",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "do it",
		RunAt:        &past,
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	sched := NewTaskSchedulerWithClock(func() time.Time { return fixedNow })
	require.NoError(t, sched.Start(context.Background()))
	defer func() { _ = sched.Stop(context.Background()) }()

	task, err := store.Database.GetChatScheduledTask(context.Background(), "task-missed")
	require.NoError(t, err)
	assert.Equal(t, string(schema.ChatScheduledTaskStateMissed), task.State)
}

func TestTaskSchedulerRegistersFutureOnceTask(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()

	base := time.Date(2026, 6, 20, 8, 0, 0, 0, time.UTC)
	runAt := base.Add(30 * time.Minute)

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-future",
		UID:          "user:alice",
		Name:         "soon",
		ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
		Prompt:       "do it",
		RunAt:        &runAt,
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	sched := NewTaskSchedulerWithClock(func() time.Time { return base })
	require.NoError(t, sched.Start(context.Background()))
	defer func() { _ = sched.Stop(context.Background()) }()

	task, err := store.Database.GetChatScheduledTask(context.Background(), "task-future")
	require.NoError(t, err)
	assert.Equal(t, string(schema.ChatScheduledTaskStateActive), task.State)
}

func TestTaskSchedulerUpdateTask(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-cron",
		UID:          "user:alice",
		Name:         "job",
		ScheduleKind: string(schema.ChatScheduledTaskKindCron),
		Cron:         "0 9 * * *",
		Prompt:       "work",
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	sched := NewTaskScheduler()
	require.NoError(t, sched.Start(context.Background()))
	defer func() { _ = sched.Stop(context.Background()) }()

	updated, err := store.Database.GetChatScheduledTask(context.Background(), "task-cron")
	require.NoError(t, err)
	updated.Cron = "0 10 * * *"
	require.NoError(t, sched.UpdateTask(updated))
}

func TestSyncTaskWithSchedulerPause(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	defer func() { store.Database = origDB }()

	require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
		Flag:         "task-pause",
		UID:          "user:alice",
		Name:         "job",
		ScheduleKind: string(schema.ChatScheduledTaskKindCron),
		Cron:         "0 9 * * *",
		Prompt:       "work",
		State:        string(schema.ChatScheduledTaskStateActive),
	}))

	sched := NewTaskScheduler()
	SetGlobalScheduler(sched)
	require.NoError(t, sched.Start(context.Background()))
	defer func() { _ = sched.Stop(context.Background()) }()

	task, err := store.Database.GetChatScheduledTask(context.Background(), "task-pause")
	require.NoError(t, err)
	paused := string(schema.ChatScheduledTaskStatePaused)
	task.State = paused
	require.NoError(t, syncTaskWithScheduler(task))

	sched.mu.Lock()
	_, registered := sched.cronIDs["task-pause"]
	sched.mu.Unlock()
	assert.False(t, registered)
}
