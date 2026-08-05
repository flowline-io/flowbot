package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestHubStore_SaveHomelabApps(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	hs := NewHubStore(client)
	ctx := context.Background()

	require.NoError(t, hs.SaveHomelabApps(ctx, []homelab.App{
		{Name: "karakeep", Path: "/apps/karakeep", Status: homelab.AppStatusRunning},
		{Name: "archivebox", Path: "/apps/archivebox", Status: homelab.AppStatusStopped},
	}))
	infos, err := hs.ListApps(ctx)
	require.NoError(t, err)
	require.Len(t, infos, 2)

	require.NoError(t, hs.SaveHomelabApps(ctx, []homelab.App{
		{Name: "karakeep", Path: "/apps/karakeep-v2", Status: homelab.AppStatusPartial},
		{Name: "immich", Path: "/apps/immich", Status: homelab.AppStatusRunning},
	}))
	infos, err = hs.ListApps(ctx)
	require.NoError(t, err)
	require.Len(t, infos, 3)

	rows, err := client.App.Query().All(ctx)
	require.NoError(t, err)
	byName := map[string]*gen.App{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.Equal(t, "/apps/karakeep-v2", byName["karakeep"].Path)
	assert.Equal(t, string(homelab.AppStatusPartial), byName["karakeep"].Status)
	assert.Equal(t, "/apps/immich", byName["immich"].Path)
}

func TestHubStore_SaveHomelabAppsKeepsLastDuplicateName(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	hs := NewHubStore(client)
	ctx := context.Background()

	require.NoError(t, hs.SaveHomelabApps(ctx, []homelab.App{
		{Name: "karakeep", Path: "/apps/old", Status: homelab.AppStatusStopped},
		{Name: "karakeep", Path: "/apps/new", Status: homelab.AppStatusRunning},
	}))
	rows, err := client.App.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "/apps/new", rows[0].Path)
	assert.Equal(t, string(homelab.AppStatusRunning), rows[0].Status)
}
