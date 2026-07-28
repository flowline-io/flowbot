package life_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/life"
)

func TestStatsWindowBounds(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 7, 28, 20, 30, 0, 0, loc)

	start, end := life.StatsWindowBounds(now, loc, 30)
	assert.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, loc), start)
	assert.Equal(t, time.Date(2026, 7, 29, 0, 0, 0, 0, loc), end)
}

func TestBuildStatsPage(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 7, 28, 18, 0, 0, 0, loc)

	tests := []struct {
		name string
		in   life.StatsInput
		check func(t *testing.T, page life.StatsPage)
	}{
		{
			name: "empty window still has 30 zero days",
			in: life.StatsInput{
				Location:     loc,
				Now:          now,
				TimezoneName: "Asia/Shanghai",
			},
			check: func(t *testing.T, page life.StatsPage) {
				assert.Equal(t, "Asia/Shanghai", page.Timezone)
				assert.Len(t, page.DayLabels, 30)
				assert.Equal(t, "2026-06-29", page.DayLabels[0])
				assert.Equal(t, "2026-07-28", page.DayLabels[29])
				assert.Equal(t, 0, page.Completions)
				assert.Equal(t, 0, page.GoldNet)
				assert.False(t, page.HasActivity)
				assert.Equal(t, []int{0, 0, 0}, page.QuestTypeValues)
				require.Len(t, page.GrowthLabels, 9)
				assert.Equal(t, life.StatsOtherBucket, page.GrowthLabels[8])
			},
		},
		{
			name: "buckets by local day and attributes growth",
			in: life.StatsInput{
				Location:     loc,
				Now:          now,
				TimezoneName: "Asia/Shanghai",
				Actions: []life.StatsActionEvent{
					{
						At:                 time.Date(2026, 7, 28, 1, 0, 0, 0, loc),
						GainedExp:          10,
						GainedGold:         5,
						Dropped:            true,
						CharacteristicCode: "INT",
					},
					{
						At:         time.Date(2026, 7, 28, 2, 0, 0, 0, loc),
						GainedExp:  7,
						GainedGold: 1,
					},
					{
						// UTC evening is next calendar day in CST — still in window as 07-28 local? 
						// 2026-07-27 20:00 UTC = 2026-07-28 04:00 CST
						At:                 time.Date(2026, 7, 27, 20, 0, 0, 0, time.UTC),
						GainedExp:          3,
						GainedGold:         2,
						CharacteristicCode: "WIL",
					},
					{
						// outside window
						At:         time.Date(2026, 6, 1, 12, 0, 0, 0, loc),
						GainedExp:  100,
						GainedGold: 100,
					},
				},
				Quests: []life.StatsQuestCompletion{
					{CompletedAt: time.Date(2026, 7, 28, 10, 0, 0, 0, loc), Type: "Daily"},
					{CompletedAt: time.Date(2026, 7, 27, 10, 0, 0, 0, loc), Type: "Boss"},
					{CompletedAt: time.Date(2026, 7, 26, 10, 0, 0, 0, loc), Type: "One-Time"},
					{CompletedAt: time.Date(2026, 7, 25, 10, 0, 0, 0, loc), Type: "Weird"},
				},
				Redemptions: []life.StatsRedemptionEvent{
					{At: time.Date(2026, 7, 28, 12, 0, 0, 0, loc), PricePaid: 4},
				},
				Unlocks: []life.StatsUnlockEvent{
					{At: time.Date(2026, 7, 20, 12, 0, 0, 0, loc)},
					{At: time.Date(2026, 5, 1, 12, 0, 0, 0, loc)},
				},
			},
			check: func(t *testing.T, page life.StatsPage) {
				assert.Equal(t, 3, page.Completions)
				assert.Equal(t, 20, page.TotalExp)
				assert.Equal(t, 8, page.GoldInTotal)
				assert.Equal(t, 4, page.GoldOutTotal)
				assert.Equal(t, 4, page.GoldNet)
				assert.True(t, page.HasActivity)
				assert.Equal(t, 1, page.AchievementUnlocks)

				last := len(page.DayLabels) - 1
				assert.Equal(t, 3, page.ActivityCounts[last]) // all three in-window actions land on 07-28 CST
				assert.Equal(t, 20, page.ActivityExp[last])
				assert.Equal(t, 8, page.GoldIn[last])
				assert.Equal(t, 4, page.GoldOut[last])
				assert.Equal(t, 1, page.Drops[last])

				assert.Equal(t, []int{1, 1, 1}, page.QuestTypeValues)

				growth := map[string]int{}
				for i, label := range page.GrowthLabels {
					growth[label] = page.GrowthValues[i]
				}
				assert.Equal(t, 10, growth["INT"])
				assert.Equal(t, 3, growth["WIL"])
				assert.Equal(t, 7, growth[life.StatsOtherBucket])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page := life.BuildStatsPage(tt.in)
			tt.check(t, page)
		})
	}
}
