package life

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

func TestCreateReward_Validation(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	tests := []struct {
		name    string
		in      CreateRewardInput
		wantErr string
	}{
		{name: "empty name", in: CreateRewardInput{Name: "  ", Price: 10}, wantErr: "life: reward name required"},
		{name: "zero price", in: CreateRewardInput{Name: "Coffee", Price: 0}, wantErr: "life: reward price must be at least 1"},
		{name: "negative cooldown", in: CreateRewardInput{Name: "Coffee", Price: 5, CooldownHours: -1}, wantErr: "life: reward cooldown cannot be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreateReward(ctx, "reward-val-"+tt.name, tt.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRedeemReward_HappyPathAndGuards(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := store.NewLifeStore(client)
	svc := NewService(ls)
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "reward-redeem-user", "", "")
	require.NoError(t, err)
	require.NoError(t, ls.SetProfileGold(ctx, profile.ID, 100))

	reward, err := svc.CreateReward(ctx, "reward-redeem-user", CreateRewardInput{
		Name: "Milk tea", Notes: "weekend only", Price: 40, CooldownHours: 24,
	})
	require.NoError(t, err)

	t.Run("insufficient gold", func(t *testing.T) {
		cheap, err := svc.CreateReward(ctx, "reward-redeem-user", CreateRewardInput{
			Name: "Yacht day", Price: 9999,
		})
		require.NoError(t, err)
		err = svc.RedeemReward(ctx, "reward-redeem-user", cheap.Flag)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient gold")
	})

	t.Run("redeem success", func(t *testing.T) {
		err := svc.RedeemReward(ctx, "reward-redeem-user", reward.Flag)
		require.NoError(t, err)

		updated, err := ls.GetProfileByID(ctx, profile.ID)
		require.NoError(t, err)
		assert.Equal(t, 60, updated.Gold)

		page, err := svc.ListRewardsPage(ctx, "reward-redeem-user")
		require.NoError(t, err)
		require.Len(t, page.Redemptions, 1)
		assert.Equal(t, "Milk tea", page.Redemptions[0].RewardName)
		assert.Equal(t, 40, page.Redemptions[0].PricePaid)
		var milk *RewardView
		for i := range page.Active {
			if page.Active[i].Flag == reward.Flag {
				milk = &page.Active[i]
				break
			}
		}
		require.NotNil(t, milk)
		assert.True(t, milk.OnCooldown)
	})

	t.Run("cooldown blocks second redeem", func(t *testing.T) {
		err := svc.RedeemReward(ctx, "reward-redeem-user", reward.Flag)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "on cooldown")
	})

	t.Run("inactive blocks redeem", func(t *testing.T) {
		require.NoError(t, svc.DeactivateReward(ctx, "reward-redeem-user", reward.Flag))
		err := svc.RedeemReward(ctx, "reward-redeem-user", reward.Flag)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
	})

	t.Run("restore keeps cooldown", func(t *testing.T) {
		require.NoError(t, svc.RestoreReward(ctx, "reward-redeem-user", reward.Flag))
		page, err := svc.ListRewardsPage(ctx, "reward-redeem-user")
		require.NoError(t, err)
		var found bool
		for _, item := range page.Active {
			if item.Flag == reward.Flag {
				found = true
				assert.True(t, item.OnCooldown)
			}
		}
		assert.True(t, found)
	})

	t.Run("price snapshot survives rename", func(t *testing.T) {
		require.NoError(t, ls.MarkRewardRedeemed(ctx, reward.ID, time.Now().Add(-48*time.Hour)))
		require.NoError(t, svc.UpdateReward(ctx, "reward-redeem-user", reward.Flag, CreateRewardInput{
			Name: "Fancy milk tea", Notes: "weekend only", Price: 80, CooldownHours: 24,
		}))
		require.NoError(t, ls.SetProfileGold(ctx, profile.ID, 100))
		require.NoError(t, svc.RedeemReward(ctx, "reward-redeem-user", reward.Flag))

		page, err := svc.ListRewardsPage(ctx, "reward-redeem-user")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(page.Redemptions), 2)
		assert.Equal(t, "Fancy milk tea", page.Redemptions[0].RewardName)
		assert.Equal(t, 80, page.Redemptions[0].PricePaid)
		assert.Equal(t, "Milk tea", page.Redemptions[1].RewardName)
		assert.Equal(t, 40, page.Redemptions[1].PricePaid)
	})
}

func TestParseRewardPriceAndCooldown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		price     string
		cooldown  string
		wantPrice int
		wantCD    int
		wantErr   bool
	}{
		{name: "valid", price: "50", cooldown: "12", wantPrice: 50, wantCD: 12},
		{name: "empty cooldown", price: "10", cooldown: "", wantPrice: 10, wantCD: 0},
		{name: "bad price", price: "x", cooldown: "1", wantErr: true},
		{name: "bad cooldown", price: "10", cooldown: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			price, err := ParseRewardPrice(tt.price)
			if tt.wantErr && tt.price == "x" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			cd, err := ParseRewardCooldownHours(tt.cooldown)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrice, price)
			assert.Equal(t, tt.wantCD, cd)
		})
	}
}
