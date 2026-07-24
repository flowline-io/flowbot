package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledTaskAPIFlow(t *testing.T) {
	withTestScheduleStore(t, func() {
		ctx := context.Background()
		uid := types.Uid("user:alice")

		created, err := CreateScheduledTaskForUID(ctx, uid, "sess-src", CreateScheduledTaskRequest{
			Name: "weekly", Prompt: "send report", Cron: "0 8 * * 1",
		})
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, string(schema.ChatScheduledTaskKindCron), created.ScheduleKind)
		assert.NotEmpty(t, created.TaskID)

		tests := []struct {
			name string
			run  func(*testing.T)
		}{
			{
				name: "list active tasks",
				run: func(t *testing.T) {
					tasks, err := ListScheduledTasksForUID(ctx, uid, nil)
					require.NoError(t, err)
					require.Len(t, tasks, 1)
					assert.Equal(t, "weekly", tasks[0].Name)
				},
			},
			{
				name: "get owned task",
				run: func(t *testing.T) {
					view, err := GetScheduledTaskForUID(ctx, uid, created.TaskID)
					require.NoError(t, err)
					assert.Equal(t, "send report", view.Prompt)
				},
			},
			{
				name: "patch prompt",
				run: func(t *testing.T) {
					prompt := "updated report"
					view, err := PatchScheduledTaskForUID(ctx, uid, created.TaskID, UpdateScheduledTaskRequest{Prompt: &prompt})
					require.NoError(t, err)
					assert.Equal(t, prompt, view.Prompt)
				},
			},
			{
				name: "list runs empty",
				run: func(t *testing.T) {
					runs, err := ListScheduledTaskRuns(ctx, uid, created.TaskID, 10)
					require.NoError(t, err)
					assert.Empty(t, runs)
				},
			},
			{
				name: "cancel task",
				run: func(t *testing.T) {
					require.NoError(t, CancelScheduledTaskForUID(ctx, uid, created.TaskID))
					_, err := GetScheduledTaskForUID(ctx, uid, created.TaskID)
					require.Error(t, err)
				},
			},
			{
				name: "other user cannot get task",
				run: func(t *testing.T) {
					_, err := GetScheduledTaskForUID(ctx, types.Uid("user:bob"), created.TaskID)
					require.Error(t, err)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, tt.run)
		}
	})
}

func TestRunAPIWithFakeModel(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "api run reply")

	ctx := context.Background()
	sessionID := "sess-api-run"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))

	pub := NewChannelPublisher(16)
	gate := NewConfirmGate(sessionID, pub, nil)
	state := NewAPIRunState(pub, gate)
	svc := NewService()
	require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
	t.Cleanup(func() { svc.ClearAPIRunState(sessionID, nil) })

	err := svc.RunAPI(ctx, RunRequest{SessionID: sessionID, Text: "hello api"}, &APIRunOptions{
		Publisher: pub,
		Confirm:   gate,
	})
	require.NoError(t, err)
	WaitForSessionTitleGenerationForTest()
	pub.Close()
}

func TestParseCreateScheduledTaskRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateScheduledTaskRequest
		wantErr bool
	}{
		{name: "valid cron", req: CreateScheduledTaskRequest{Name: "daily", Prompt: "check", Cron: "0 9 * * *"}},
		{name: "missing name", req: CreateScheduledTaskRequest{Prompt: "check", Cron: "0 9 * * *"}, wantErr: true},
		{name: "missing schedule", req: CreateScheduledTaskRequest{Name: "daily", Prompt: "check"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseCreateScheduledTaskRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
