package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRunStatsURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kind    string
		resName string
		days    int
		groupBy string
		want    string
	}{
		{
			name:    "global pipelines 30d day",
			kind:    "pipelines",
			days:    30,
			groupBy: "day",
			want:    "/service/web/pipelines/stats?days=30&groupBy=day",
		},
		{
			name:    "named workflow all week",
			kind:    "workflows",
			resName: "demo",
			days:    0,
			groupBy: "week",
			want:    "/service/web/workflows/demo/stats?days=0&groupBy=week",
		},
		{
			name:    "escapes workflow name",
			kind:    "workflows",
			resName: "a/b",
			days:    90,
			groupBy: "month",
			want:    "/service/web/workflows/a%2Fb/stats?days=90&groupBy=month",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildRunStatsURL(tt.kind, tt.resName, tt.days, tt.groupBy))
		})
	}
}

func TestStatsBtnClasses(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "join-item btn btn-sm btn-active", statsRangeBtnClass(30, 30))
	assert.Equal(t, "join-item btn btn-sm", statsRangeBtnClass(30, 90))
	assert.Equal(t, "join-item btn btn-sm btn-active", statsGroupByBtnClass("week", "week"))
	assert.Equal(t, "join-item btn btn-sm", statsGroupByBtnClass("day", "month"))
}
