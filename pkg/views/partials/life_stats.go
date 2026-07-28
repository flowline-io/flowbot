package partials

import (
	"github.com/bytedance/sonic"

	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// LifeStatsData is the Stats analytics HTMX panel model.
type LifeStatsData struct {
	Timezone           string
	Completions        int
	TotalExp           int
	GoldNet            int
	AchievementUnlocks int
	HasActivity        bool
	ChartsJSON         string
}

// LifeBuildStatsChartsJSON encodes chart series for life-stats.js.
func LifeBuildStatsChartsJSON(page pkglife.StatsPage) string {
	payload := map[string]any{
		"day_labels":        page.DayLabels,
		"activity_counts":   page.ActivityCounts,
		"activity_exp":      page.ActivityExp,
		"growth_labels":     page.GrowthLabels,
		"growth_values":     page.GrowthValues,
		"quest_type_labels": page.QuestTypeLabels,
		"quest_type_values": page.QuestTypeValues,
		"gold_in":           page.GoldIn,
		"gold_out":          page.GoldOut,
		"drops":             page.Drops,
	}
	b, err := sonic.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// LifeStatsFromPage maps a domain stats page into the panel view model.
func LifeStatsFromPage(page *pkglife.StatsPage) LifeStatsData {
	if page == nil {
		return LifeStatsData{Timezone: "UTC", ChartsJSON: "{}"}
	}
	return LifeStatsData{
		Timezone:           page.Timezone,
		Completions:        page.Completions,
		TotalExp:           page.TotalExp,
		GoldNet:            page.GoldNet,
		AchievementUnlocks: page.AchievementUnlocks,
		HasActivity:        page.HasActivity,
		ChartsJSON:         LifeBuildStatsChartsJSON(*page),
	}
}
