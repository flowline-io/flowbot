package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

func TestLifeStore_ListRewardsPageAndRedemptionsPage(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "rewards-page-user", "Ada", "Architect")
	require.NoError(t, err)

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		reward, err := ls.CreateReward(ctx, profile.ID, LifeRewardCreate{
			Name: fmt.Sprintf("Reward %d", i), Price: 10,
		})
		require.NoError(t, err)
		require.NoError(t, ls.SetRewardActive(ctx, reward.ID, false))
		_, err = ls.CreateRewardRedemption(ctx, profile.ID, reward.ID, reward.Name, 10, base.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	inactiveOnly := false
	page1, total, err := ls.ListRewardsPage(ctx, profile.ID, &inactiveOnly, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)
	assert.Equal(t, "Reward 4", page1[0].Name)

	page2, total, err := ls.ListRewardsPage(ctx, profile.ID, &inactiveOnly, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)
	assert.Equal(t, "Reward 2", page2[0].Name)

	logs1, logTotal, err := ls.ListRewardRedemptionsPage(ctx, profile.ID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, logTotal)
	require.Len(t, logs1, 2)
	assert.Equal(t, "Reward 4", logs1[0].RewardName)

	logs2, logTotal, err := ls.ListRewardRedemptionsPage(ctx, profile.ID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, logTotal)
	require.Len(t, logs2, 2)
	assert.Equal(t, "Reward 2", logs2[0].RewardName)
}
