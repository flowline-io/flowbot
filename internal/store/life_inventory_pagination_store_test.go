package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

func TestLifeStore_ListInventoryPage(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "inv-page-user", "Ada", "Architect")
	require.NoError(t, err)
	for i := range 5 {
		eq, err := ls.UpsertEquipment(ctx, fmt.Sprintf("eq-inv-%d", i), fmt.Sprintf("Gear %d", i), "Common", "Armor", "", nil, nil)
		require.NoError(t, err)
		_, err = ls.CreateInventory(ctx, profile.ID, eq.ID, nil, "none")
		require.NoError(t, err)
	}

	page1, total, err := ls.ListInventoryPage(ctx, profile.ID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)

	page2, total, err := ls.ListInventoryPage(ctx, profile.ID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)
	assert.NotEqual(t, page1[0].Flag, page2[0].Flag)
}
