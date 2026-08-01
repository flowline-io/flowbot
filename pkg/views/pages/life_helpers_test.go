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
