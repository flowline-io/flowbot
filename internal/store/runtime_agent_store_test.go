package store

import (
	"context"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAgentHostCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uid := types.Uid("user:agent-host")
	topic := "default"
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "create get list and update online fields",
			run: func(t *testing.T, client *gen.Client) {
				id, err := NewRuntimeAgentStore(client).CreateAgent(ctx, &gen.Agent{
					UID: uid.String(), Topic: topic, Hostid: "host-1", Hostname: "node-a",
					LastOnlineAt: now, CreatedAt: now, UpdatedAt: now,
				})
				require.NoError(t, err)
				assert.Positive(t, id)

				got, err := NewRuntimeAgentStore(client).GetAgentByHostid(ctx, uid, topic, "host-1")
				require.NoError(t, err)
				assert.Equal(t, "node-a", got.Hostname)

				all, err := NewRuntimeAgentStore(client).GetAgents(ctx)
				require.NoError(t, err)
				require.Len(t, all, 1)

				onlineAt := now.Add(5 * time.Minute)
				require.NoError(t, NewRuntimeAgentStore(client).UpdateAgentLastOnlineAt(ctx, uid, topic, "host-1", onlineAt))

				offlineAt := onlineAt.Add(2 * time.Minute)
				require.NoError(t, NewRuntimeAgentStore(client).UpdateAgentOnlineDuration(ctx, uid, topic, "host-1", offlineAt))

				updated, err := NewRuntimeAgentStore(client).GetAgentByHostid(ctx, uid, topic, "host-1")
				require.NoError(t, err)
				assert.Equal(t, onlineAt.Unix(), updated.LastOnlineAt.Unix())
				assert.Equal(t, int32(120), updated.OnlineDuration)
			},
		},
		{
			name: "get by hostid missing returns not found",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewRuntimeAgentStore(client).GetAgentByHostid(ctx, uid, topic, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "update online duration missing returns not found",
			run: func(t *testing.T, client *gen.Client) {
				err := NewRuntimeAgentStore(client).UpdateAgentOnlineDuration(ctx, uid, topic, "ghost", time.Now())
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, sqlitetest.OpenClient(t, t.Name()))
		})
	}
}
