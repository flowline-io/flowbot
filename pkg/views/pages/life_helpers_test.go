package pages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

func TestLifeHPFromStats(t *testing.T) {
	t.Parallel()
	stats := []pages.LifeStatRow{pages.LifeBuildStatRow("WIL", "Willpower", 4, 20)}
	cur, maxHP, filled, total := pages.LifeHPFromStats(stats, 2)
	assert.Equal(t, 1000, maxHP)
	assert.Equal(t, 10, total)
	assert.Positive(t, cur)
	assert.GreaterOrEqual(t, filled, 0)
}
