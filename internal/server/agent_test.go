package server

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentActionPull(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	uid := types.Uid("user-agent")
	expireAt := time.Now().UTC().Add(time.Hour)
	_, err := store.Database.CreateInstruct(ctx, &gen.Instruct{
		No: "inst-1", UID: uid.String(), Object: string(schema.InstructObjectAgent),
		Bot: "bot-a", Flag: "flag-1",
		Content: map[string]any{"text": "run backup"}, State: int(schema.InstructCreate), ExpireAt: expireAt,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		action  types.Action
		wantLen int
		wantErr bool
	}{
		{name: "pull returns pending instructs", action: types.PullAction, wantLen: 1},
		{name: "unknown action returns nil", action: types.Action("noop"), wantLen: 0},
		{name: "ack missing no errors", action: types.AckAction, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := types.AgentData{Action: tt.action, Version: types.ApiVersion}
			if tt.action == types.AckAction {
				data.Content = types.KV{}
			}
			result, err := agentAction(ctx, uid, data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantLen == 0 {
				assert.Nil(t, result)
				return
			}
			list, ok := result.([]types.KV)
			require.True(t, ok)
			assert.Len(t, list, tt.wantLen)
			assert.Equal(t, "inst-1", list[0]["no"])
		})
	}
}

func TestAgentActionAck(t *testing.T) {
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	uid := types.Uid("user-agent")
	_, err := store.Database.CreateInstruct(ctx, &gen.Instruct{
		No: "inst-ack", UID: uid.String(), Object: string(schema.InstructObjectAgent),
		Bot: "bot-a", Flag: "flag-ack", Content: map[string]any{"cmd": "run"},
		State: int(schema.InstructCreate),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		content types.KV
		wantErr bool
	}{
		{name: "valid ack updates instruct", content: types.KV{"no": "inst-ack"}},
		{name: "missing no", content: types.KV{}, wantErr: true},
		{name: "invalid no type", content: types.KV{"no": 123}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agentAction(ctx, uid, types.AgentData{
				Action: types.AckAction, Version: types.ApiVersion, Content: tt.content,
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAgentActionOnlineOfflineMessage(t *testing.T) {
	setupTestCacheStore(t)

	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = origDB })

	ctx := context.Background()
	uid := types.Uid("user-agent")

	tests := []struct {
		name    string
		action  types.Action
		content types.KV
		wantErr bool
	}{
		{
			name:    "online missing hostid",
			action:  types.OnlineAction,
			content: types.KV{"hostname": "node-a"},
			wantErr: true,
		},
		{
			name:    "offline updates duration",
			action:  types.OfflineAction,
			content: types.KV{"hostid": "host-1", "hostname": "node-a"},
		},
		{
			name:    "message forwards payload",
			action:  types.MessageAction,
			content: types.KV{"message": "disk full"},
		},
		{
			name:    "message missing body",
			action:  types.MessageAction,
			content: types.KV{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agentAction(ctx, uid, types.AgentData{
				Action: tt.action, Version: types.ApiVersion, Content: tt.content,
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
