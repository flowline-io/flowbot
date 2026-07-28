package life

import (
	"time"
)

// StatsWindowDays is the fixed analytics window length.
const StatsWindowDays = 30

// StatsOtherBucket is the growth chart label for unattributed EXP.
const StatsOtherBucket = "Other"

// StatsActionEvent is one action-log row for stats aggregation.
type StatsActionEvent struct {
	At                 time.Time
	GainedExp          int
	GainedGold         int
	Dropped            bool
	CharacteristicCode string // empty → Other
}

// StatsQuestCompletion is one completed quest for type breakdown.
type StatsQuestCompletion struct {
	CompletedAt time.Time
	Type        string
}

// StatsRedemptionEvent is one reward redemption for gold outflow.
type StatsRedemptionEvent struct {
	At        time.Time
	PricePaid int
}

// StatsUnlockEvent is one achievement unlock in the window.
type StatsUnlockEvent struct {
	At time.Time
}

// StatsInput is the raw event set for BuildStatsPage.
type StatsInput struct {
	Actions      []StatsActionEvent
	Quests       []StatsQuestCompletion
	Redemptions  []StatsRedemptionEvent
	Unlocks      []StatsUnlockEvent
	Location     *time.Location
	Now          time.Time
	TimezoneName string
}

// StatsPage is the aggregated Life stats dashboard model.
type StatsPage struct {
	Timezone           string
	DayLabels          []string
	Completions        int
	TotalExp           int
	GoldInTotal        int
	GoldOutTotal       int
	GoldNet            int
	ActivityCounts     []int
	ActivityExp        []int
	GrowthLabels       []string
	GrowthValues       []int
	QuestTypeLabels    []string
	QuestTypeValues    []int
	GoldIn             []int
	GoldOut            []int
	Drops              []int
	AchievementUnlocks int
	HasActivity        bool
	WindowStart        time.Time
	WindowEndExclusive time.Time
}

// StatsWindowBounds returns [start, endExclusive) for the last days calendar days in loc,
// ending at the start of tomorrow relative to now.
func StatsWindowBounds(now time.Time, loc *time.Location, days int) (start, endExclusive time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	if days < 1 {
		days = StatsWindowDays
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	endExclusive = today.AddDate(0, 0, 1)
	start = today.AddDate(0, 0, -(days - 1))
	return start, endExclusive
}

// BuildStatsPage aggregates action / quest / redemption / unlock events into chart series.
func BuildStatsPage(in StatsInput) StatsPage {
	loc := in.Location
	if loc == nil {
		loc = time.UTC
	}
	tz := in.TimezoneName
	if tz == "" {
		tz = loc.String()
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	start, endExclusive := StatsWindowBounds(now, loc, StatsWindowDays)
	labels, index := statsDayIndex(start, endExclusive)

	n := len(labels)
	activityCounts := make([]int, n)
	activityExp := make([]int, n)
	goldIn := make([]int, n)
	goldOut := make([]int, n)
	drops := make([]int, n)
	growth := newGrowthBuckets()

	completions, totalExp, goldInTotal := foldActions(in.Actions, loc, start, endExclusive, index, activityCounts, activityExp, goldIn, drops, growth)
	goldOutTotal := foldRedemptions(in.Redemptions, loc, start, endExclusive, index, goldOut)
	questTypes, questValues := foldQuestTypes(in.Quests, start, endExclusive)
	growthLabels, growthValues := growthSeries(growth)
	unlocks := countUnlocks(in.Unlocks, start, endExclusive)

	return StatsPage{
		Timezone:           tz,
		DayLabels:          labels,
		Completions:        completions,
		TotalExp:           totalExp,
		GoldInTotal:        goldInTotal,
		GoldOutTotal:       goldOutTotal,
		GoldNet:            goldInTotal - goldOutTotal,
		ActivityCounts:     activityCounts,
		ActivityExp:        activityExp,
		GrowthLabels:       growthLabels,
		GrowthValues:       growthValues,
		QuestTypeLabels:    questTypes,
		QuestTypeValues:    questValues,
		GoldIn:             goldIn,
		GoldOut:            goldOut,
		Drops:              drops,
		AchievementUnlocks: unlocks,
		HasActivity:        completions > 0 || goldOutTotal > 0 || unlocks > 0 || sumInts(questValues) > 0,
		WindowStart:        start,
		WindowEndExclusive: endExclusive,
	}
}

func statsDayIndex(start, endExclusive time.Time) ([]string, map[string]int) {
	labels := make([]string, 0, StatsWindowDays)
	index := make(map[string]int, StatsWindowDays)
	for d := start; d.Before(endExclusive); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		index[key] = len(labels)
		labels = append(labels, key)
	}
	return labels, index
}

func newGrowthBuckets() map[string]int {
	growth := make(map[string]int, len(DefaultCharacteristics)+1)
	for _, c := range DefaultCharacteristics {
		growth[c.Code] = 0
	}
	growth[StatsOtherBucket] = 0
	return growth
}

func foldActions(
	actions []StatsActionEvent,
	loc *time.Location,
	start, endExclusive time.Time,
	index map[string]int,
	activityCounts, activityExp, goldIn, drops []int,
	growth map[string]int,
) (completions, totalExp, goldInTotal int) {
	for _, a := range actions {
		i, ok := dayIndex(a.At, loc, start, endExclusive, index)
		if !ok {
			continue
		}
		completions++
		totalExp += a.GainedExp
		goldInTotal += a.GainedGold
		activityCounts[i]++
		activityExp[i] += a.GainedExp
		goldIn[i] += a.GainedGold
		if a.Dropped {
			drops[i]++
		}
		growth[resolveGrowthCode(a.CharacteristicCode, growth)] += a.GainedExp
	}
	return completions, totalExp, goldInTotal
}

func foldRedemptions(
	redemptions []StatsRedemptionEvent,
	loc *time.Location,
	start, endExclusive time.Time,
	index map[string]int,
	goldOut []int,
) int {
	total := 0
	for _, r := range redemptions {
		i, ok := dayIndex(r.At, loc, start, endExclusive, index)
		if !ok {
			continue
		}
		total += r.PricePaid
		goldOut[i] += r.PricePaid
	}
	return total
}

func foldQuestTypes(quests []StatsQuestCompletion, start, endExclusive time.Time) (labels []string, values []int) {
	labels = []string{"Daily", "One-Time", "Boss"}
	counts := map[string]int{"Daily": 0, "One-Time": 0, "Boss": 0}
	for _, q := range quests {
		if !inWindow(q.CompletedAt, start, endExclusive) {
			continue
		}
		if _, ok := counts[q.Type]; ok {
			counts[q.Type]++
		}
	}
	values = make([]int, len(labels))
	for i, t := range labels {
		values[i] = counts[t]
	}
	return labels, values
}

func growthSeries(growth map[string]int) (labels []string, values []int) {
	labels = make([]string, 0, len(DefaultCharacteristics)+1)
	values = make([]int, 0, len(DefaultCharacteristics)+1)
	for _, c := range DefaultCharacteristics {
		labels = append(labels, c.Code)
		values = append(values, growth[c.Code])
	}
	labels = append(labels, StatsOtherBucket)
	values = append(values, growth[StatsOtherBucket])
	return labels, values
}

func countUnlocks(unlocks []StatsUnlockEvent, start, endExclusive time.Time) int {
	n := 0
	for _, u := range unlocks {
		if inWindow(u.At, start, endExclusive) {
			n++
		}
	}
	return n
}

func resolveGrowthCode(code string, growth map[string]int) string {
	if code == "" {
		return StatsOtherBucket
	}
	if _, known := growth[code]; !known {
		return StatsOtherBucket
	}
	return code
}

func dayIndex(at time.Time, loc *time.Location, start, endExclusive time.Time, index map[string]int) (int, bool) {
	if !inWindow(at, start, endExclusive) {
		return 0, false
	}
	i, ok := index[at.In(loc).Format("2006-01-02")]
	return i, ok
}

func inWindow(at, start, endExclusive time.Time) bool {
	if at.IsZero() {
		return false
	}
	return !at.Before(start) && at.Before(endExclusive)
}

func sumInts(vals []int) int {
	n := 0
	for _, v := range vals {
		n += v
	}
	return n
}
