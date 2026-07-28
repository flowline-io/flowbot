package life

import "maps"

import "strings"

const (
	// AchievementKindFirst is a one-shot memorial (threshold treated as 1).
	AchievementKindFirst = "first"
	// AchievementKindCount is a cumulative memorial with visible progress.
	AchievementKindCount = "count"

	achievementKeyPrefix = "quest_completed"
	achievementAny       = "*"
)

// AchievementDef is one catalog achievement definition (pure data).
type AchievementDef struct {
	Flag        string
	Name        string
	Description string
	Active      bool
	Kind        string
	QuestType   string
	Difficulty  string
	Threshold   int
	SortOrder   int
}

// AchievementConditionKey builds the progress key for optional type/difficulty filters.
// Empty questType or difficulty means "any" (*).
func AchievementConditionKey(questType, difficulty string) string {
	return achievementKeyPrefix + ":" + normalizeAchievementFilter(questType) + ":" + normalizeAchievementFilter(difficulty)
}

// AchievementKeysForCompletion returns every progress key bumped by one quest completion.
func AchievementKeysForCompletion(questType, difficulty string) []string {
	qt := strings.TrimSpace(questType)
	diff := strings.TrimSpace(difficulty)
	keys := []string{
		AchievementConditionKey("", ""),
	}
	if qt != "" {
		keys = append(keys, AchievementConditionKey(qt, ""))
	}
	if diff != "" {
		keys = append(keys, AchievementConditionKey("", diff))
	}
	if qt != "" && diff != "" {
		keys = append(keys, AchievementConditionKey(qt, diff))
	}
	return keys
}

// AchievementEvalInput is the pure input for evaluating unlocks after a quest completion.
type AchievementEvalInput struct {
	QuestType  string
	Difficulty string
	Catalog    []AchievementDef
	Progress   map[string]int
	Unlocked   map[string]struct{}
}

// AchievementEvalResult is the progress deltas and newly unlocked definitions.
type AchievementEvalResult struct {
	ProgressAfter map[string]int
	NewlyUnlocked []AchievementDef
}

// EvaluateAchievements bumps matching condition keys and returns new unlocks.
func EvaluateAchievements(in AchievementEvalInput) AchievementEvalResult {
	progress := cloneIntMap(in.Progress)
	bumped := AchievementKeysForCompletion(in.QuestType, in.Difficulty)
	for _, key := range bumped {
		progress[key]++
	}
	unlocked := in.Unlocked
	if unlocked == nil {
		unlocked = map[string]struct{}{}
	}
	var newly []AchievementDef
	for _, def := range in.Catalog {
		if !def.Active {
			continue
		}
		if _, ok := unlocked[def.Flag]; ok {
			continue
		}
		threshold := achievementThreshold(def)
		if threshold <= 0 {
			continue
		}
		key := AchievementConditionKey(def.QuestType, def.Difficulty)
		if progress[key] < threshold {
			continue
		}
		newly = append(newly, def)
	}
	after := make(map[string]int, len(bumped))
	for _, key := range bumped {
		after[key] = progress[key]
	}
	return AchievementEvalResult{ProgressAfter: after, NewlyUnlocked: newly}
}

// AchievementShowsProgress reports whether the UI should show N/M for this definition.
func AchievementShowsProgress(def AchievementDef) bool {
	return achievementThreshold(def) > 1
}

func achievementThreshold(def AchievementDef) int {
	kind := strings.TrimSpace(def.Kind)
	if kind == "" {
		if def.Threshold > 1 {
			kind = AchievementKindCount
		} else {
			kind = AchievementKindFirst
		}
	}
	switch kind {
	case AchievementKindFirst:
		return 1
	case AchievementKindCount:
		if def.Threshold < 1 {
			return 1
		}
		return def.Threshold
	default:
		if def.Threshold < 1 {
			return 1
		}
		return def.Threshold
	}
}

func normalizeAchievementFilter(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return achievementAny
	}
	return v
}

func cloneIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	maps.Copy(out, in)
	return out
}
