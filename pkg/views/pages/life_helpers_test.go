package pages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestLifeBuildStatRowAndRadar(t *testing.T) {
	t.Parallel()
	row := pages.LifeBuildStatRow("INT", "Intelligence", 3, 150)
	assert.Equal(t, 24, row.TotalSegs)
	assert.InDelta(t, 3.5, row.RadarValue, 0.001)
	// Within-level bar: 150/300 → 50% → 12/24 segments.
	assert.Equal(t, 12, row.FilledSegs)
	assert.Equal(t, int64(150), row.Exp)
	assert.Equal(t, int64(300), row.ExpToNext)

	high := pages.LifeBuildStatRow("INT", "Intelligence", 20, 50)
	assert.InDelta(t, 20.025, high.RadarValue, 0.001) // 50/2000
	assert.Greater(t, high.RadarValue, 10.0)

	writing := pages.LifeBuildStatRow("WRI", "Writing", 1, 75)
	assert.InDelta(t, 1.75, writing.RadarValue, 0.001)
	assert.Equal(t, 18, writing.FilledSegs)

	empty := pages.LifeBuildStatRow("PHY", "Physique", 1, 0)
	assert.InDelta(t, 1.0, empty.RadarValue, 0.001)
	assert.Equal(t, 0, empty.FilledSegs)

	labels, values := pages.LifeMarshalRadar([]pages.LifeStatRow{row})
	assert.Contains(t, labels, "Intelligence")
	assert.Contains(t, values, "3.5")
}

func TestLifeDisplayNameAndClassTraits(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Ada", pages.LifeDisplayName("Ada", "user-admin"))
	assert.Equal(t, "admin", pages.LifeDisplayName("", "user-admin"))
	assert.Equal(t, "Creative", pages.LifeClassStrength("Architect"))
	assert.Equal(t, "Impatient", pages.LifeClassWeakness("Architect"))
}

func TestLifeBuildEquipSlots(t *testing.T) {
	t.Parallel()
	empty := pages.LifeBuildEquipSlots(nil)
	assert.Len(t, empty, 6)
	assert.Equal(t, "Head", empty[0].Label)
	assert.Equal(t, "head_slot", empty[0].SlotField)
	assert.Nil(t, empty[0].Item)
	assert.Equal(t, "Artifact", empty[5].Label)

	filled := pages.LifeBuildEquipSlots([]pages.LifeInventoryRow{
		{Flag: "inv-1", Name: "Steady Hoodie", Rarity: "Common", Slot: "Armor", SlotField: "armor_slot", Equipped: true},
		{Flag: "inv-2", Name: "Spare Hoodie", Rarity: "Common", Slot: "Armor", SlotField: "", Equipped: false},
		{Flag: "inv-3", Name: "Focus Ring", Rarity: "Rare", Slot: "Accessory", SlotField: "accessory_slot", Equipped: true, Tarnished: true},
	})
	assert.Len(t, filled, 6)
	assert.Nil(t, filled[0].Item)
	requireSlot := func(label string) pages.LifeEquipSlot {
		t.Helper()
		for _, s := range filled {
			if s.Label == label {
				return s
			}
		}
		t.Fatalf("missing slot %s", label)
		return pages.LifeEquipSlot{}
	}
	armor := requireSlot("Armor")
	assert.NotNil(t, armor.Item)
	assert.Equal(t, "Steady Hoodie", armor.Item.Name)
	assert.Equal(t, "armor_slot", armor.SlotField)
	acc := requireSlot("Accessory")
	assert.NotNil(t, acc.Item)
	assert.True(t, acc.Item.Tarnished)
	assert.Equal(t, "Focus Ring", acc.Item.Name)
}

func TestLifeRarityClass(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "rarity-common", pages.LifeRarityClass("Common"))
	assert.Equal(t, "rarity-uncommon", pages.LifeRarityClass("Uncommon"))
	assert.Equal(t, "rarity-rare", pages.LifeRarityClass("Rare"))
	assert.Equal(t, "rarity-epic", pages.LifeRarityClass("Epic"))
	assert.Equal(t, "rarity-legendary", pages.LifeRarityClass("Legendary"))
	assert.Equal(t, "rarity-mythic", pages.LifeRarityClass("Mythic"))
	assert.Equal(t, "rarity-common", pages.LifeRarityClass(""))
	assert.Empty(t, pages.LifeSlotRarityClass(pages.LifeEquipSlot{}))
	assert.Equal(t, "rarity-rare", pages.LifeSlotRarityClass(pages.LifeEquipSlot{
		Item: &pages.LifeInventoryRow{Rarity: "Rare"},
	}))
}

func TestLifeVerdictChipClass(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "life-meta-chip-ok", pages.LifeVerdictChipClass("completed"))
	assert.Equal(t, "life-meta-chip-diff", pages.LifeVerdictChipClass("partial"))
	assert.Equal(t, "life-meta-chip-danger", pages.LifeVerdictChipClass("failed"))
	assert.Equal(t, "life-meta-chip-warn", pages.LifeVerdictChipClass("needs_more_evidence"))
	assert.Empty(t, pages.LifeVerdictChipClass(""))
}

func TestLifeHPFromStats(t *testing.T) {
	t.Parallel()
	stats := []pages.LifeStatRow{pages.LifeBuildStatRow("WIL", "Willpower", 4, 20)}
	cur, maxHP, filled, total := pages.LifeHPFromStats(stats, 2)
	assert.Equal(t, pkglife.SoftHPMax, maxHP)
	assert.Equal(t, pkglife.SoftHPHeartCount, total)
	assert.Positive(t, cur)
	assert.GreaterOrEqual(t, filled, 0)
}

func TestLifeFormatBuffText(t *testing.T) {
	t.Parallel()
	got := pages.LifeFormatBuffText(map[string]any{
		"INT":      float64(5),
		"DropRate": float64(0.02),
		"GoldMult": float64(0.1),
	})
	assert.Contains(t, got, "INT +5")
	assert.Contains(t, got, "Drop +2%")
	assert.Contains(t, got, "Gold +10%")
	assert.Empty(t, pages.LifeFormatBuffText(nil))
}

func TestLifeFormatPerkText(t *testing.T) {
	t.Parallel()
	got := pages.LifeFormatPerkText(map[string]any{
		"ai_breakdown_depth": "deep",
	})
	assert.Equal(t, "ai_breakdown_depth=deep", got)
	assert.Empty(t, pages.LifeFormatPerkText(nil))
}

func TestLifePlanLabelsAndIndent(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Goal", pages.LifePlanNodeTypeLabel("goal"))
	assert.Equal(t, "Action", pages.LifePlanNodeTypeLabel("action"))
	assert.Equal(t, "Habit (pending)", pages.LifeTaskTypeLabel("habit_candidate"))
	assert.Equal(t, "Habit", pages.LifeTaskTypeLabel("habit"))
	assert.Equal(t, "Checkpoint", pages.LifeTaskTypeLabel("checkpoint"))
	assert.Equal(t, "Action", pages.LifeActionLogSourceLabel("occurrence"))
	assert.Equal(t, "Habit", pages.LifeActionLogSourceLabel("habit_checkin"))
	assert.Equal(t, "Checkpoint", pages.LifeActionLogSourceLabel("checkpoint"))
	assert.Equal(t, "One-time", pages.LifeOccurrenceKindLabel("one_time"))
	assert.Equal(t, "Recurring", pages.LifeOccurrenceKindLabel("recurring"))
	assert.Empty(t, pages.LifeIndentStyle(0))
	assert.Equal(t, "margin-left:2rem;", pages.LifeIndentStyle(2))
}

func TestLifeBuildPageInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		page, perPage, total int
		wantPage, wantPages  int
		wantPrev, wantNext   bool
	}{
		{name: "empty", page: 1, perPage: 10, total: 0, wantPage: 1, wantPages: 0},
		{name: "first of many", page: 1, perPage: 10, total: 25, wantPage: 1, wantPages: 3, wantNext: true},
		{name: "middle", page: 2, perPage: 10, total: 25, wantPage: 2, wantPages: 3, wantPrev: true, wantNext: true},
		{name: "last", page: 3, perPage: 10, total: 25, wantPage: 3, wantPages: 3, wantPrev: true},
		{name: "clamp high page", page: 9, perPage: 10, total: 25, wantPage: 3, wantPages: 3, wantPrev: true},
		{name: "normalize low page", page: 0, perPage: 10, total: 5, wantPage: 1, wantPages: 1},
		{name: "default per page", page: 1, perPage: 0, total: 11, wantPage: 1, wantPages: 2, wantNext: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pages.LifeBuildPageInfo(tt.page, tt.perPage, tt.total)
			assert.Equal(t, tt.wantPage, got.Page)
			assert.Equal(t, tt.wantPages, got.TotalPages)
			assert.Equal(t, tt.total, got.Total)
			assert.Equal(t, tt.wantPrev, got.HasPrev)
			assert.Equal(t, tt.wantNext, got.HasNext)
		})
	}
}

func TestLifeQuestsListURLAndPagers(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "completed", pages.LifeNormalizeHistoryTab(""))
	assert.Equal(t, "logs", pages.LifeNormalizeHistoryTab("logs"))
	assert.Equal(t, "life-tab is-active", pages.LifeHistoryTabClass("logs", "logs"))
	assert.Equal(t, "life-tab", pages.LifeHistoryTabClass("logs", "completed"))

	assert.Equal(t, "/service/web/life/quests", pages.LifeQuestsListURL(1, 1, "", ""))
	assert.Equal(t, "/service/web/life/quests?completed_page=2#life-history", pages.LifeQuestsListURL(2, 1, pages.LifeHistoryTabCompleted, pages.LifeHistoryAnchor))
	assert.Equal(t, "/service/web/life/quests?history_tab=logs&logs_page=3#life-history", pages.LifeQuestsListURL(1, 3, pages.LifeHistoryTabActionLogs, pages.LifeHistoryAnchor))
	assert.Equal(t, "/service/web/life/quests?completed_page=2&logs_page=3#life-history", pages.LifeQuestsListURL(2, 3, pages.LifeHistoryTabCompleted, pages.LifeHistoryAnchor))

	completed := pages.LifeWithCompletedPager(pages.LifeBuildPageInfo(2, 10, 25), 3)
	assert.Equal(t, "/service/web/life/quests?logs_page=3#life-history", completed.PrevURL)
	assert.Equal(t, "/service/web/life/quests?completed_page=3&logs_page=3#life-history", completed.NextURL)

	logs := pages.LifeWithActionLogsPager(pages.LifeBuildPageInfo(2, 10, 25), 4)
	assert.Equal(t, "/service/web/life/quests?completed_page=4&history_tab=logs#life-history", logs.PrevURL)
	assert.Equal(t, "/service/web/life/quests?completed_page=4&history_tab=logs&logs_page=3#life-history", logs.NextURL)
}
