package partials

import (
	"context"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/i18n"
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
func LifeBuildStatsChartsJSON(ctx context.Context, page pkglife.StatsPage) string {
	payload := map[string]any{
		"day_labels":        page.DayLabels,
		"activity_counts":   page.ActivityCounts,
		"activity_exp":      page.ActivityExp,
		"growth_labels":     lifeStatsGrowthLabels(ctx, page.GrowthLabels),
		"growth_values":     page.GrowthValues,
		"quest_type_labels": lifeStatsQuestTypeLabels(ctx, page.QuestTypeLabels),
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
func LifeStatsFromPage(ctx context.Context, page *pkglife.StatsPage) LifeStatsData {
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
		ChartsJSON:         LifeBuildStatsChartsJSON(ctx, *page),
	}
}

func lifeStatsGrowthLabels(ctx context.Context, labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, len(labels))
	for i, label := range labels {
		if label == pkglife.StatsOtherBucket {
			out[i] = i18n.T(ctx, "life.stat.other")
			continue
		}
		key := "life.stat." + strings.ToLower(label)
		msg := i18n.T(ctx, key)
		if msg != key {
			out[i] = msg
			continue
		}
		out[i] = label
	}
	return out
}

func lifeStatsQuestTypeLabels(ctx context.Context, labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, len(labels))
	for i, label := range labels {
		key := "life.quest_type." + strings.ToLower(strings.ReplaceAll(label, "-", "_"))
		msg := i18n.T(ctx, key)
		if msg != key {
			out[i] = msg
			continue
		}
		out[i] = label
	}
	return out
}
