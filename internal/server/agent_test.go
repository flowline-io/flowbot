package server

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

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
