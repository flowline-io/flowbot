package life

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/notify"
)

func TestFormatQuestCompletedSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result *CompleteResult
		want   string
	}{
		{
			name: "rewards only",
			result: &CompleteResult{
				GainedExp: 12, GainedGold: 5,
			},
			want: "+12 EXP · +5 gold",
		},
		{
			name: "drop with rarity",
			result: &CompleteResult{
				GainedExp: 10, GainedGold: 3,
				Dropped: true, ItemName: "Iron Sword", ItemRarity: "Rare",
			},
			want: "+10 EXP · +3 gold\nDropped: Iron Sword (Rare)",
		},
		{
			name: "profile level up wins over skill",
			result: &CompleteResult{
				GainedExp: 20, GainedGold: 8,
				ProfileLevelBefore: 3, ProfileLevelAfter: 4,
				SkillName: "Exploration", SkillLevelBefore: 2, SkillLevelAfter: 3,
			},
			want: "+20 EXP · +8 gold\nLevel up: Profile Lv 3 → 4",
		},
		{
			name: "skill level up when profile unchanged",
			result: &CompleteResult{
				GainedExp: 8, GainedGold: 2,
				ProfileLevelBefore: 5, ProfileLevelAfter: 5,
				SkillName: "Exploration", SkillLevelBefore: 2, SkillLevelAfter: 3,
			},
			want: "+8 EXP · +2 gold\nLevel up: Exploration Lv 2 → 3",
		},
		{
			name: "achievements appended",
			result: &CompleteResult{
				GainedExp: 1, GainedGold: 1,
				NewlyUnlocked: []UnlockedAchievement{
					{Flag: "first", Name: "First Blood"},
					{Flag: "owl", Name: "Night Owl"},
				},
			},
			want: "+1 EXP · +1 gold\nAchievements: First Blood, Night Owl",
		},
		{
			name: "full composite",
			result: &CompleteResult{
				GainedExp: 50, GainedGold: 20,
				Dropped: true, ItemName: "Crown", ItemRarity: "Legendary",
				ProfileLevelBefore: 9, ProfileLevelAfter: 10,
				NewlyUnlocked: []UnlockedAchievement{{Name: "Decathlete"}},
			},
			want: "+50 EXP · +20 gold\nDropped: Crown (Legendary)\nLevel up: Profile Lv 9 → 10\nAchievements: Decathlete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatQuestCompletedSummary(tt.result))
		})
	}
}

func TestBuildQuestCompletedPayload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		result      *CompleteResult
		wantTitle   string
		wantURL     string
		wantSummary string
	}{
		{
			name: "no drop links to quests",
			result: &CompleteResult{
				Quest:     &gen.LifeQuest{Title: "Ship the feature"},
				GainedExp: 4, GainedGold: 1,
			},
			wantTitle:   "Quest completed · Ship the feature",
			wantURL:     lifeInboxQuestsURL,
			wantSummary: "+4 EXP · +1 gold",
		},
		{
			name: "drop links to inventory",
			result: &CompleteResult{
				Quest:     &gen.LifeQuest{Title: "Boss fight"},
				GainedExp: 4, GainedGold: 1,
				Dropped: true, ItemName: "Relic", ItemRarity: "Epic",
			},
			wantTitle:   "Quest completed · Boss fight",
			wantURL:     lifeInboxInventoryURL,
			wantSummary: "+4 EXP · +1 gold\nDropped: Relic (Epic)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildQuestCompletedPayload(tt.result)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantTitle, got[notify.PayloadKeyTitle])
			assert.Equal(t, tt.wantURL, got[notify.PayloadKeyURL])
			assert.Equal(t, tt.wantSummary, got[notify.PayloadKeySummary])
		})
	}
}

func TestBuildQuestFailedPayload(t *testing.T) {
	t.Parallel()
	got := buildQuestFailedPayload("Daily stretch")
	assert.Equal(t, "Quest failed · Daily stretch", got[notify.PayloadKeyTitle])
	assert.Equal(t, "Equipment tarnished for 24h.", got[notify.PayloadKeySummary])
	assert.Equal(t, lifeInboxQuestsURL, got[notify.PayloadKeyURL])
}
