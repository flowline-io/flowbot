package life_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/life"
)

func TestAchievementConditionKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		questType  string
		difficulty string
		want       string
	}{
		{name: "any any", want: "quest_completed:*:*"},
		{name: "daily any", questType: "Daily", want: "quest_completed:Daily:*"},
		{name: "any ss", difficulty: "SS", want: "quest_completed:*:SS"},
		{name: "boss sss", questType: "Boss", difficulty: "SSS", want: "quest_completed:Boss:SSS"},
		{name: "trim spaces", questType: " Daily ", difficulty: " S ", want: "quest_completed:Daily:S"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, life.AchievementConditionKey(tt.questType, tt.difficulty))
		})
	}
}

func TestAchievementKeysForCompletion(t *testing.T) {
	t.Parallel()
	keys := life.AchievementKeysForCompletion("Daily", "SS")
	assert.Equal(t, []string{
		"quest_completed:*:*",
		"quest_completed:Daily:*",
		"quest_completed:*:SS",
		"quest_completed:Daily:SS",
	}, keys)
}

func TestEvaluateAchievements(t *testing.T) {
	t.Parallel()
	catalog := []life.AchievementDef{
		{Flag: "ach-first", Name: "First Steps", Active: true, Kind: life.AchievementKindFirst, Threshold: 1},
		{Flag: "ach-daily-10", Name: "Daily Ten", Active: true, Kind: life.AchievementKindCount, QuestType: "Daily", Threshold: 10},
		{Flag: "ach-ss", Name: "SS Once", Active: true, Kind: life.AchievementKindFirst, Difficulty: "SS", Threshold: 1},
		{Flag: "ach-retired", Name: "Retired", Active: false, Kind: life.AchievementKindFirst, Threshold: 1},
	}

	t.Run("first unlock and progress bump", func(t *testing.T) {
		t.Parallel()
		res := life.EvaluateAchievements(life.AchievementEvalInput{
			QuestType:  "Daily",
			Difficulty: "C",
			Catalog:    catalog,
			Progress:   map[string]int{},
		})
		require.Len(t, res.NewlyUnlocked, 1)
		assert.Equal(t, "ach-first", res.NewlyUnlocked[0].Flag)
		assert.Equal(t, 1, res.ProgressAfter["quest_completed:*:*"])
		assert.Equal(t, 1, res.ProgressAfter["quest_completed:Daily:*"])
		assert.Equal(t, 1, res.ProgressAfter["quest_completed:*:C"])
		assert.Equal(t, 1, res.ProgressAfter["quest_completed:Daily:C"])
	})

	t.Run("cumulative unlock at threshold", func(t *testing.T) {
		t.Parallel()
		res := life.EvaluateAchievements(life.AchievementEvalInput{
			QuestType:  "Daily",
			Difficulty: "B",
			Catalog:    catalog,
			Progress: map[string]int{
				"quest_completed:*:*":     20,
				"quest_completed:Daily:*": 9,
			},
			Unlocked: map[string]struct{}{"ach-first": {}},
		})
		require.Len(t, res.NewlyUnlocked, 1)
		assert.Equal(t, "ach-daily-10", res.NewlyUnlocked[0].Flag)
		assert.Equal(t, 10, res.ProgressAfter["quest_completed:Daily:*"])
	})

	t.Run("skips already unlocked and inactive", func(t *testing.T) {
		t.Parallel()
		res := life.EvaluateAchievements(life.AchievementEvalInput{
			QuestType:  "One-Time",
			Difficulty: "SS",
			Catalog:    catalog,
			Progress:   map[string]int{},
			Unlocked:   map[string]struct{}{"ach-first": {}, "ach-ss": {}},
		})
		assert.Empty(t, res.NewlyUnlocked)
		assert.Equal(t, 1, res.ProgressAfter["quest_completed:*:SS"])
	})
}

func TestAchievementShowsProgress(t *testing.T) {
	t.Parallel()
	assert.False(t, life.AchievementShowsProgress(life.AchievementDef{Kind: life.AchievementKindFirst, Threshold: 1}))
	assert.True(t, life.AchievementShowsProgress(life.AchievementDef{Kind: life.AchievementKindCount, Threshold: 10}))
}
