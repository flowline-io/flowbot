// Package life provides pure formulas for Flowbot Life (cascade, loot, buffs, achievements).
package life

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"strings"
	"time"
)

// DefaultCharacteristics are seeded on EnsureProfile (and backfilled for older profiles).
var DefaultCharacteristics = []struct {
	Code string
	Name string
}{
	{Code: "INT", Name: "Intelligence"},
	{Code: "PHY", Name: "Physique"},
	{Code: "WIL", Name: "Willpower"},
	{Code: "CHA", Name: "Charisma"},
	{Code: "CRE", Name: "Creativity"},
	{Code: "FIN", Name: "Financial"},
	{Code: "WRI", Name: "Writing"},
	{Code: "FOC", Name: "Focus"},
}

// PARA goal categories (life_goals.category).
const (
	GoalCategoryProject  = "Project"
	GoalCategoryArea     = "Area"
	GoalCategoryResource = "Resource"
)

// Goal lifecycle statuses (life_goals.status).
const (
	GoalStatusActive    = "Active"
	GoalStatusPaused    = "Paused"
	GoalStatusCompleted = "Completed"
)

// ExpToNextLevel returns exp required to advance from the given level.
func ExpToNextLevel(level int) int64 {
	if level < 1 {
		level = 1
	}
	return int64(100 * level)
}

// StatSnapshot is mutable progress for cascade application.
type StatSnapshot struct {
	Level      int
	CurrentExp int64
}

// CascadeInput is the reward applied when a quest completes.
type CascadeInput struct {
	BaseExp                  int
	BaseGold                 int
	Skill                    StatSnapshot
	Characteristic           StatSnapshot
	Profile                  StatSnapshot
	ProfileGold              int
	ExpToCharacteristicRatio float64
}

// CascadeResult is the post-cascade state and gained totals.
type CascadeResult struct {
	Skill          StatSnapshot
	Characteristic StatSnapshot
	Profile        StatSnapshot
	ProfileGold    int
	GainedExp      int
	GainedGold     int
}

// DefaultRewards returns fear, exp, gold, and drop tier for a difficulty code.
// Everyday tiers (E–S) grow ~1.5x with gold near exp/3; SS/SSS jump harder for
// boss-tier payoff (C still anchors ~4 completions to ExpToNextLevel(1)=100).
func DefaultRewards(diff string) (fear, exp, gold int, tier string) {
	switch NormalizeDifficulty(diff) {
	case "SSS":
		return 5, 350, 110, "Mythic"
	case "SS":
		return 5, 220, 70, "Legendary"
	case "S":
		return 4, 100, 30, "Epic"
	case "A":
		return 3, 65, 20, "Epic"
	case "B":
		return 3, 40, 12, "Rare"
	case "D":
		return 1, 15, 5, "Common"
	case "E":
		return 1, 10, 3, "Common"
	default: // C
		return 2, 25, 8, "Common"
	}
}

// NormalizeDifficulty coerces a raw difficulty label to a known tier.
func NormalizeDifficulty(raw string) string {
	d := strings.ToUpper(strings.TrimSpace(raw))
	switch d {
	case "SSS", "SS", "S", "A", "B", "C", "D", "E":
		return d
	default:
		return "C"
	}
}

// ApplyCascade distributes quest rewards into skill, characteristic, and profile snapshots.
func ApplyCascade(in CascadeInput) CascadeResult {
	ratio := in.ExpToCharacteristicRatio
	if ratio <= 0 {
		ratio = 0.5
	}
	skillExp := int64(in.BaseExp)
	charExp := int64(math.Round(float64(in.BaseExp) * ratio))

	skill := applyExp(in.Skill, skillExp)
	characteristic := applyExp(in.Characteristic, charExp)
	profile := applyExp(in.Profile, skillExp)

	return CascadeResult{
		Skill:          skill,
		Characteristic: characteristic,
		Profile:        profile,
		ProfileGold:    in.ProfileGold + in.BaseGold,
		GainedExp:      in.BaseExp,
		GainedGold:     in.BaseGold,
	}
}

func applyExp(s StatSnapshot, gained int64) StatSnapshot {
	if s.Level < 1 {
		s.Level = 1
	}
	s.CurrentExp += gained
	for {
		need := ExpToNextLevel(s.Level)
		if s.CurrentExp < need {
			break
		}
		s.CurrentExp -= need
		s.Level++
	}
	return s
}

// LootInput drives a single drop roll.
type LootInput struct {
	BaseDropChance   float64
	ProfileBonus     float64
	EquippedDropRate float64
	PityCount        int
	PityThreshold    int
	Roll             float64 // [0,1); if < 0, caller should set via RollUnit
	PoolSize         int
}

// LootResult is the outcome of a loot roll.
type LootResult struct {
	Dropped    bool
	ForcedPity bool
	Roll       float64
	Threshold  float64
	PoolIndex  int
	NextPity   int
}

// DefaultPityThreshold is the miss count that forces a drop.
const DefaultPityThreshold = 10

// ResolveLoot decides whether an item drops and which pool index.
func ResolveLoot(in LootInput) LootResult {
	pityThreshold := in.PityThreshold
	if pityThreshold <= 0 {
		pityThreshold = DefaultPityThreshold
	}
	roll := in.Roll
	if roll < 0 {
		roll = 0
	}
	threshold := in.BaseDropChance + in.ProfileBonus + in.EquippedDropRate
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}

	forced := in.PityCount+1 >= pityThreshold
	dropped := forced || roll <= threshold
	nextPity := 0
	if !dropped {
		nextPity = in.PityCount + 1
	}

	idx := 0
	if in.PoolSize > 0 {
		idx = int(roll * float64(in.PoolSize))
		if idx >= in.PoolSize {
			idx = in.PoolSize - 1
		}
		if idx < 0 {
			idx = 0
		}
	}

	return LootResult{
		Dropped:    dropped && in.PoolSize > 0,
		ForcedPity: forced && dropped,
		Roll:       roll,
		Threshold:  threshold,
		PoolIndex:  idx,
		NextPity:   nextPity,
	}
}

// PreviewDropChance returns the effective drop threshold without rolling.
func PreviewDropChance(in LootInput) float64 {
	threshold := in.BaseDropChance + in.ProfileBonus + in.EquippedDropRate
	if threshold < 0 {
		return 0
	}
	if threshold > 1 {
		return 1
	}
	return threshold
}

// RustDuration is how long equipped gear stays tarnished after a failed quest.
const RustDuration = 24 * time.Hour

// IsTarnished reports whether rust is still active at now.
func IsTarnished(until *time.Time, now time.Time) bool {
	return until != nil && until.After(now)
}

// RollUnit returns a cryptographically strong unit interval in [0,1).
func RollUnit() (float64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	// 53-bit mantissa → uniform [0,1), matching math/rand.Float64 spacing.
	u := binary.LittleEndian.Uint64(b[:]) >> 11
	return float64(u) / (1 << 53), nil
}

// BuffTotals aggregates equipped item buffs.
type BuffTotals struct {
	Stats    map[string]float64
	DropRate float64
	GoldMult float64
}

// MergeBuffs merges template and instance buff maps (instance wins on key clash).
func MergeBuffs(template, instance map[string]any) map[string]float64 {
	out := map[string]float64{}
	apply := func(m map[string]any) {
		for k, v := range m {
			switch n := v.(type) {
			case float64:
				out[k] = n
			case float32:
				out[k] = float64(n)
			case int:
				out[k] = float64(n)
			case int64:
				out[k] = float64(n)
			case int32:
				out[k] = float64(n)
			}
		}
	}
	apply(template)
	apply(instance)
	return out
}

// SumEquippedBuffs sums float buffs from multiple items.
func SumEquippedBuffs(items []map[string]float64) BuffTotals {
	totals := BuffTotals{Stats: map[string]float64{}, GoldMult: 1}
	for _, item := range items {
		for k, v := range item {
			switch k {
			case "DropRate":
				totals.DropRate += v
			case "GoldMult":
				totals.GoldMult += v
			default:
				totals.Stats[k] += v
			}
		}
	}
	if totals.GoldMult < 0 {
		totals.GoldMult = 0
	}
	return totals
}

// SlotField maps slot_type to equipped_slots column name.
func SlotField(slotType string) string {
	switch slotType {
	case "Head":
		return "head_slot"
	case "Weapon":
		return "weapon_slot"
	case "Armor":
		return "armor_slot"
	case "Shoes":
		return "shoes_slot"
	case "Accessory":
		return "accessory_slot"
	case "Artifact":
		return "artifact_slot"
	default:
		return ""
	}
}

// NeedsInstanceLore reports whether a drop should enqueue memorial lore generation.
func NeedsInstanceLore(questType, difficulty string) bool {
	return questType == "Boss" || difficulty == "SSS" || difficulty == "SS"
}

const (
	// SoftHPMax is the soft HP ceiling used by the character HUD.
	SoftHPMax = 1000
	// SoftHPHeartCount is how many heart segments the HUD shows.
	SoftHPHeartCount = 10
)

// SoftHPFromWillpower derives a soft HP pool from Willpower (or profile level fallback).
func SoftHPFromWillpower(wilLevel int, wilExp, wilNeed int64, profileLevel int) (current, maxHP int) {
	maxHP = SoftHPMax
	if wilLevel < 1 {
		wilLevel = profileLevel
	}
	if wilLevel < 1 {
		wilLevel = 1
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
	return current, maxHP
}

// BlendCompletionRate applies a 90/10 exponential blend after a quest outcome.
func BlendCompletionRate(rate float64, success bool) float64 {
	sample := 0.0
	if success {
		sample = 1.0
	}
	return rate*0.9 + sample*0.1
}

// SkillTreeWindowDays maps a cadence hint to an activity window length in days.
func SkillTreeWindowDays(cadence string) int {
	value := strings.ToLower(strings.TrimSpace(cadence))
	switch {
	case strings.Contains(value, "daily"), strings.Contains(value, "day"):
		return 7
	case strings.Contains(value, "week"):
		return 21
	case strings.Contains(value, "month"):
		return 45
	default:
		return 14
	}
}

// SourceTypeLabel returns a concise label for one audit / skill-tree source.
func SourceTypeLabel(sourceType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "habit_checkin":
		return "Habit"
	case "occurrence":
		return "Action"
	case "checkpoint":
		return "Checkpoint"
	default:
		return "Quest"
	}
}

// TaskTypeLabel returns a concise label for one plan action type.
func TaskTypeLabel(taskType string) string {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "todo":
		return "Todo"
	case "recurring":
		return "Recurring"
	case "habit_candidate":
		return "Habit (pending)"
	case "habit":
		return "Habit"
	case "checkpoint":
		return "Checkpoint"
	default:
		return ""
	}
}
