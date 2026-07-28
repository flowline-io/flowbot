package life_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/life"
)

func TestApplyCascade(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         life.CascadeInput
		wantSkill  int64
		wantChar   int64
		wantGold   int
		wantLevels int // profile level after
	}{
		{
			name: "basic no level up",
			in: life.CascadeInput{
				BaseExp: 50, BaseGold: 10,
				Skill:                    life.StatSnapshot{Level: 1, CurrentExp: 0},
				Characteristic:           life.StatSnapshot{Level: 1, CurrentExp: 0},
				Profile:                  life.StatSnapshot{Level: 1, CurrentExp: 0},
				ProfileGold:              0,
				ExpToCharacteristicRatio: 0.5,
			},
			wantSkill: 50, wantChar: 25, wantGold: 10, wantLevels: 1,
		},
		{
			name: "skill levels up",
			in: life.CascadeInput{
				BaseExp: 150, BaseGold: 5,
				Skill:                    life.StatSnapshot{Level: 1, CurrentExp: 0},
				Characteristic:           life.StatSnapshot{Level: 1, CurrentExp: 0},
				Profile:                  life.StatSnapshot{Level: 1, CurrentExp: 0},
				ExpToCharacteristicRatio: 0.5,
			},
			wantSkill: 50, wantChar: 75, wantGold: 5, wantLevels: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := life.ApplyCascade(tt.in)
			assert.Equal(t, tt.wantGold, got.ProfileGold)
			assert.Equal(t, tt.wantSkill, got.Skill.CurrentExp)
			assert.Equal(t, tt.wantChar, got.Characteristic.CurrentExp)
			assert.Equal(t, tt.wantLevels, got.Profile.Level)
		})
	}
}

func TestResolveLoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		in          life.LootInput
		wantDropped bool
		wantPity    int
	}{
		{
			name: "miss increments pity",
			in: life.LootInput{
				BaseDropChance: 0.1, Roll: 0.9, PoolSize: 3, PityCount: 2, PityThreshold: 10,
			},
			wantDropped: false, wantPity: 3,
		},
		{
			name: "hit drops",
			in: life.LootInput{
				BaseDropChance: 0.5, Roll: 0.1, PoolSize: 3, PityCount: 0, PityThreshold: 10,
			},
			wantDropped: true, wantPity: 0,
		},
		{
			name: "pity forces drop",
			in: life.LootInput{
				BaseDropChance: 0.01, Roll: 0.99, PoolSize: 2, PityCount: 9, PityThreshold: 10,
			},
			wantDropped: true, wantPity: 0,
		},
		{
			name: "empty pool never drops",
			in: life.LootInput{
				BaseDropChance: 1, Roll: 0, PoolSize: 0, PityCount: 9, PityThreshold: 10,
			},
			wantDropped: false, wantPity: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := life.ResolveLoot(tt.in)
			assert.Equal(t, tt.wantDropped, got.Dropped)
			assert.Equal(t, tt.wantPity, got.NextPity)
		})
	}
}

func TestMergeAndSumBuffs(t *testing.T) {
	t.Parallel()
	merged := life.MergeBuffs(
		map[string]any{"INT": 5, "DropRate": 0.05},
		map[string]any{"INT": 10, "GoldMult": 0.2},
	)
	require.InDelta(t, 10, merged["INT"], 0.001)
	require.InDelta(t, 0.05, merged["DropRate"], 0.001)

	totals := life.SumEquippedBuffs([]map[string]float64{merged, {"PHY": 3, "DropRate": 0.02}})
	assert.InDelta(t, 0.07, totals.DropRate, 0.001)
	assert.InDelta(t, 10, totals.Stats["INT"], 0.001)
	assert.InDelta(t, 3, totals.Stats["PHY"], 0.001)
	assert.InDelta(t, 1.2, totals.GoldMult, 0.001)
}

func TestSlotField(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "weapon_slot", life.SlotField("Weapon"))
	assert.Empty(t, life.SlotField("Unknown"))
}

func TestExpToNextLevel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(100), life.ExpToNextLevel(1))
	assert.Equal(t, int64(200), life.ExpToNextLevel(2))
}

func TestDefaultRewards(t *testing.T) {
	t.Parallel()
	fear, exp, gold, tier := life.DefaultRewards("A")
	assert.Equal(t, 3, fear)
	assert.Equal(t, 65, exp)
	assert.Equal(t, 20, gold)
	assert.Equal(t, "Epic", tier)
	fear, exp, gold, tier = life.DefaultRewards("SS")
	assert.Equal(t, 5, fear)
	assert.Equal(t, 220, exp)
	assert.Equal(t, 70, gold)
	assert.Equal(t, "Legendary", tier)
	fear, exp, gold, tier = life.DefaultRewards("SSS")
	assert.Equal(t, 5, fear)
	assert.Equal(t, 350, exp)
	assert.Equal(t, 110, gold)
	assert.Equal(t, "Mythic", tier)
	_, exp, gold, _ = life.DefaultRewards("unknown")
	assert.Equal(t, 25, exp)
	assert.Equal(t, 8, gold)
	assert.Equal(t, "C", life.NormalizeDifficulty("z"))
}

func TestDefaultCharacteristics(t *testing.T) {
	t.Parallel()
	require.Len(t, life.DefaultCharacteristics, 8)
	codes := make([]string, 0, len(life.DefaultCharacteristics))
	for _, c := range life.DefaultCharacteristics {
		codes = append(codes, c.Code)
	}
	assert.Equal(t, []string{"INT", "PHY", "WIL", "CHA", "CRE", "FIN", "WRI", "FOC"}, codes)
}

func TestPreviewDropChanceAndTarnish(t *testing.T) {
	t.Parallel()
	chance := life.PreviewDropChance(life.LootInput{
		BaseDropChance: 0.2, ProfileBonus: 0.05, EquippedDropRate: 0.1,
	})
	assert.InDelta(t, 0.35, chance, 0.001)
	assert.InDelta(t, 0.0, life.PreviewDropChance(life.LootInput{BaseDropChance: -1}), 0.001)
	until := time.Now().Add(life.RustDuration)
	assert.True(t, life.IsTarnished(&until, time.Now()))
	assert.False(t, life.IsTarnished(nil, time.Now()))
}

func TestSoftHPBlendAndLabels(t *testing.T) {
	t.Parallel()
	cur, maxHP := life.SoftHPFromWillpower(4, 20, 400, 2)
	assert.Equal(t, life.SoftHPMax, maxHP)
	assert.Equal(t, 400+4*50+int((20*50)/400), cur)
	assert.InDelta(t, 0.91, life.BlendCompletionRate(0.9, true), 0.001)
	assert.InDelta(t, 0.81, life.BlendCompletionRate(0.9, false), 0.001)
	assert.Equal(t, 7, life.SkillTreeWindowDays("daily"))
	assert.Equal(t, 21, life.SkillTreeWindowDays("weekly"))
	assert.True(t, life.NeedsInstanceLore("Boss", "A"))
	assert.True(t, life.NeedsInstanceLore("One-Time", "SS"))
	assert.False(t, life.NeedsInstanceLore("Daily", "B"))
	assert.Equal(t, "Habit", life.SourceTypeLabel("habit_checkin"))
	assert.Equal(t, "Habit (pending)", life.TaskTypeLabel("habit_candidate"))
}
