package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetScheduledTaskStateForUID(t *testing.T) {
	withTestScheduleStore(t, func() {
		require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
			Flag:         "task-set-state",
			UID:          "user:alice",
			Name:         "daily",
			ScheduleKind: string(schema.ChatScheduledTaskKindCron),
			Cron:         "0 9 * * *",
			Prompt:       "check logs",
			State:        string(schema.ChatScheduledTaskStateActive),
		}))

		tests := []struct {
			name      string
			uid       types.Uid
			taskID    string
			state     string
			wantState string
			wantErr   error
		}{
			{
				name:      "pause active task",
				uid:       types.Uid("user:alice"),
				taskID:    "task-set-state",
				state:     string(schema.ChatScheduledTaskStatePaused),
				wantState: string(schema.ChatScheduledTaskStatePaused),
			},
			{
				name:    "reject invalid state",
				uid:     types.Uid("user:alice"),
				taskID:  "task-set-state",
				state:   "archived",
				wantErr: types.ErrInvalidArgument,
			},
			{
				name:    "not found for other user",
				uid:     types.Uid("user:bob"),
				taskID:  "task-set-state",
				state:   string(schema.ChatScheduledTaskStateCancelled),
				wantErr: types.ErrNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				view, err := SetScheduledTaskStateForUID(context.Background(), tt.uid, tt.taskID, tt.state)
				if tt.wantErr != nil {
					require.Error(t, err)
					assert.ErrorIs(t, err, tt.wantErr)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, view)
				assert.Equal(t, tt.wantState, view.State)

				row, getErr := store.Database.GetChatScheduledTask(context.Background(), tt.taskID)
				require.NoError(t, getErr)
				assert.Equal(t, tt.wantState, row.State)
			})
		}
	})
}

func TestSetScheduledTaskStateForUIDCompletedTask(t *testing.T) {
	withTestScheduleStore(t, func() {
		runAt := time.Now().UTC().Add(2 * time.Hour)
		require.NoError(t, store.Database.CreateChatScheduledTask(context.Background(), &gen.ChatScheduledTask{
			Flag:         "task-completed",
			UID:          "user:alice",
			Name:         "once",
			ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
			Prompt:       "run once",
			RunAt:        &runAt,
			NextRunAt:    &runAt,
			State:        string(schema.ChatScheduledTaskStateCompleted),
		}))

		view, err := SetScheduledTaskStateForUID(
			context.Background(),
			types.Uid("user:alice"),
			"task-completed",
			string(schema.ChatScheduledTaskStateActive),
		)
		require.NoError(t, err)
		require.NotNil(t, view)
		assert.Equal(t, string(schema.ChatScheduledTaskStateActive), view.State)
	})
}
