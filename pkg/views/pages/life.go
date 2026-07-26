package pages

import (
	"fmt"
	"math"
	"strings"

	"github.com/bytedance/sonic"

	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// LifeDashboardData is the Life home dashboard model (Character Identity layout).
type LifeDashboardData = LifeCharacterData

// LifeStatRow is one characteristic row with progress for segmented bars.
type LifeStatRow struct {
	Code       string
	Name       string
	Level      int
	Exp        int64
	ExpToNext  int64
	FilledSegs int
	TotalSegs  int
	RadarValue float64
}

// LifeSkillRow is one skill row.
type LifeSkillRow struct {
	Name  string
	Level int
	Exp   int64
}

// LifeCharacterData is the Character Identity page model.
type LifeCharacterData struct {
	Nickname        string
	ClassType       string
	Level           int
	Exp             int64
	ExpToNext       int64
	LevelFilledSegs int
	LevelTotalSegs  int
	Gold            int
	HPCurrent       int
	HPMax           int
	HeartsFilled    int
	HeartsTotal     int
	MasterObjective string
	MinorObjective  string
	Strength        string
	Weakness        string
	Characteristics []LifeStatRow
	Skills          []LifeSkillRow
	DropRateBonus   float64
	GoldMult        float64
	PendingCount    int
	RadarLabelsJSON string
	RadarValuesJSON string
	ShowClassForm   bool
	Goals           []LifeGoalRow
}

// LifeGoalRow is one PARA goal for Character / Quests UI.
type LifeGoalRow struct {
	Flag     string
	Title    string
	Category string
	Status   string
}

// LifeQuestRow is one quest list row.
type LifeQuestRow struct {
	Flag       string
	Title      string
	Prompt     string
	Type       string
	Difficulty string
	Fear       int
	Exp        int
	Gold       int
	DropTier   string
	DropChance float64
	Status     string
}

// LifeQuestsData is the quests page model.
type LifeQuestsData struct {
	Pending      []LifeQuestRow
	Completed    []LifeQuestRow
	Goals        []LifeGoalRow
	ActionLogs   []LifeActionLogRow
	PendingCount int
}

// LifeActionLogRow is one completion audit row.
type LifeActionLogRow struct {
	Flag       string
	QuestTitle string
	GainedExp  int
	GainedGold int
	Dice       float64
	HasDice    bool
	Dropped    bool
	When       string
}

// LifeInventoryRow is one backpack row.
type LifeInventoryRow struct {
	Flag       string
	Name       string
	Rarity     string
	Slot       string
	SlotField  string
	Lore       string
	LoreStatus string
	Equipped   bool
	Tarnished  bool
}

// LifeEquipSlot is one wear-position cell on the inventory equipment board.
type LifeEquipSlot struct {
	Label     string
	SlotField string
	Item      *LifeInventoryRow
}

// LifeInventoryData is the inventory page model.
type LifeInventoryData struct {
	Slots        []LifeEquipSlot
	Items        []LifeInventoryRow
	PendingCount int
}

// lifeEquipSlotOrder is the paper-doll display order for wear positions.
var lifeEquipSlotOrder = []struct {
	label string
	field string
}{
	{"Head", "head_slot"},
	{"Weapon", "weapon_slot"},
	{"Armor", "armor_slot"},
	{"Shoes", "shoes_slot"},
	{"Accessory", "accessory_slot"},
	{"Artifact", "artifact_slot"},
}

// LifeBuildEquipSlots builds the six wear-position cells from backpack rows.
// Empty positions keep a labeled empty cell; equipped items fill their slot.
func LifeBuildEquipSlots(items []LifeInventoryRow) []LifeEquipSlot {
	byField := make(map[string]LifeInventoryRow, len(items))
	for _, it := range items {
		if !it.Equipped || it.SlotField == "" {
			continue
		}
		byField[it.SlotField] = it
	}
	out := make([]LifeEquipSlot, 0, len(lifeEquipSlotOrder))
	for _, def := range lifeEquipSlotOrder {
		cell := LifeEquipSlot{Label: def.label, SlotField: def.field}
		if it, ok := byField[def.field]; ok {
			cp := it
			cell.Item = &cp
		}
		out = append(out, cell)
	}
	return out
}

func lifeEquippedCount(slots []LifeEquipSlot) string {
	n := 0
	for _, s := range slots {
		if s.Item != nil {
			n++
		}
	}
	return fmt.Sprintf("%d / %d", n, len(slots))
}

// LifeRarityClass maps equipment rarity to a CSS modifier class.
func LifeRarityClass(rarity string) string {
	switch strings.ToLower(strings.TrimSpace(rarity)) {
	case "common":
		return "rarity-common"
	case "uncommon":
		return "rarity-uncommon"
	case "rare":
		return "rarity-rare"
	case "epic":
		return "rarity-epic"
	case "legendary":
		return "rarity-legendary"
	case "mythic":
		return "rarity-mythic"
	default:
		return "rarity-common"
	}
}

// LifeSlotRarityClass returns the rarity CSS class for a filled equip slot.
func LifeSlotRarityClass(slot LifeEquipSlot) string {
	if slot.Item == nil {
		return ""
	}
	return LifeRarityClass(slot.Item.Rarity)
}

const lifeSegmentCount = 24
const lifeHeartCount = 10
const lifeHPMax = 1000

// LifeBuildStatRow builds a characteristic row with segment fill.
// Bar fill and "exp / need" are within the current level (levels have no cap).
// RadarValue is absolute progress (level + in-level fraction) with no upper bound.
func LifeBuildStatRow(code, name string, level int, exp int64) LifeStatRow {
	if level < 1 {
		level = 1
	}
	need := pkglife.ExpToNextLevel(level)
	filled := lifeFilledSegments(exp, need, lifeSegmentCount)
	return LifeStatRow{
		Code: code, Name: name, Level: level, Exp: exp, ExpToNext: need,
		FilledSegs: filled, TotalSegs: lifeSegmentCount,
		RadarValue: lifeAbsoluteProgress(level, exp, need),
	}
}

func lifeAbsoluteProgress(level int, exp, need int64) float64 {
	radar := float64(level)
	if need > 0 && exp > 0 {
		frac := float64(exp) / float64(need)
		if frac > 1 {
			frac = 1
		}
		radar += frac
	}
	return radar
}

func lifeFilledFromRatio(ratio float64, segments int) int {
	if segments <= 0 || ratio <= 0 {
		return 0
	}
	if ratio >= 1 {
		return segments
	}
	n := int(math.Round(ratio * float64(segments)))
	if n < 1 {
		return 1
	}
	if n > segments {
		return segments
	}
	return n
}

func lifeFilledSegments(current, ceiling int64, segments int) int {
	if ceiling <= 0 {
		return 0
	}
	return lifeFilledFromRatio(float64(current)/float64(ceiling), segments)
}

// LifeDisplayName picks a short display name from nickname or uid.
func LifeDisplayName(nickname, uid string) string {
	n := strings.TrimSpace(nickname)
	if n != "" {
		return n
	}
	u := strings.TrimPrefix(uid, "user:")
	u = strings.TrimPrefix(u, "user-")
	if u == "" {
		return "Operator"
	}
	return u
}

// LifeHPFromStats derives a soft HP pool from Willpower (or average level).
func LifeHPFromStats(stats []LifeStatRow, profileLevel int) (current, maxHP, heartsFilled, heartsTotal int) {
	maxHP = lifeHPMax
	heartsTotal = lifeHeartCount
	wilLevel := profileLevel
	wilExp := int64(0)
	wilNeed := pkglife.ExpToNextLevel(profileLevel)
	for _, s := range stats {
		if s.Code == "WIL" {
			wilLevel = s.Level
			wilExp = s.Exp
			wilNeed = s.ExpToNext
			break
		}
	}
	current = 400 + wilLevel*50
	if wilNeed > 0 {
		current += int((wilExp * 50) / wilNeed)
	}
	if current > maxHP {
		current = maxHP
	}
	if current < 0 {
		current = 0
	}
	heartsFilled = lifeFilledSegments(int64(current), int64(maxHP), heartsTotal)
	return current, maxHP, heartsFilled, heartsTotal
}

// LifeClassStrength maps class to a strength label.
func LifeClassStrength(classType string) string {
	switch classType {
	case "Mage":
		return "Analytical"
	case "Warrior":
		return "Resilient"
	case "Rogue":
		return "Adaptive"
	default:
		return "Creative"
	}
}

// LifeClassWeakness maps class to a mild weakness label.
func LifeClassWeakness(classType string) string {
	switch classType {
	case "Mage":
		return "Overthinking"
	case "Warrior":
		return "Stubborn"
	case "Rogue":
		return "Restless"
	default:
		return "Impatient"
	}
}

// LifeMarshalRadar encodes radar chart labels/values as JSON arrays.
func LifeMarshalRadar(stats []LifeStatRow) (labelsJSON, valuesJSON string) {
	labels := make([]string, 0, len(stats))
	values := make([]float64, 0, len(stats))
	for _, s := range stats {
		labels = append(labels, s.Name)
		values = append(values, s.RadarValue)
	}
	lb, err := sonic.Marshal(labels)
	if err != nil {
		return "[]", "[]"
	}
	vb, err := sonic.Marshal(values)
	if err != nil {
		return "[]", "[]"
	}
	return string(lb), string(vb)
}

// LifeSegmentIndexes returns 0..n-1 for templ loops.
func LifeSegmentIndexes(n int) []int {
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}
