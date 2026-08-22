package partials_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/i18n"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestLifeBuildStatsChartsJSON(t *testing.T) {
	t.Parallel()
	page := pkglife.BuildStatsPage(pkglife.StatsInput{
		Location:     time.UTC,
		Now:          time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC),
		TimezoneName: "UTC",
		Actions: []pkglife.StatsActionEvent{
			{At: time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), GainedExp: 5, GainedGold: 2},
		},
	})
	ctx := i18n.DefaultContext()
	raw := partials.LifeBuildStatsChartsJSON(ctx, page)
	assert.Contains(t, raw, `"day_labels"`)
	assert.Contains(t, raw, `"activity_counts"`)
	assert.Contains(t, raw, `"growth_labels"`)
}

func TestLifeStatsFromPageNil(t *testing.T) {
	t.Parallel()
	got := partials.LifeStatsFromPage(i18n.DefaultContext(), nil)
	assert.Equal(t, "UTC", got.Timezone)
	assert.Equal(t, "{}", got.ChartsJSON)
}
