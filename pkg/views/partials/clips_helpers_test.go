package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatClipListTime(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 7, 17, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{name: "zero", t: time.Time{}, want: ""},
		{name: "utc stamp", t: ts, want: "2026-07-17 11:00 UTC"},
		{name: "non-utc converted", t: ts.In(time.FixedZone("CST", 8*3600)), want: "2026-07-17 11:00 UTC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatClipListTime(tt.t))
		})
	}
}
