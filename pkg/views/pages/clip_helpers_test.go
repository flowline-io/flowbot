package pages

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClipPageTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    ClipPageData
		want string
	}{
		{name: "not found", d: ClipPageData{NotFound: true}, want: "Clip not found"},
		{name: "empty title", d: ClipPageData{}, want: "Clip"},
		{name: "with title", d: ClipPageData{Title: "Hello"}, want: "Hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, clipPageTitle(tt.d))
		})
	}
}

func TestFormatClipMeta(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 7, 17, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		createdAt time.Time
		words     int
		want      string
	}{
		{name: "full meta", createdAt: ts, words: 629, want: "Jul 17, 2026, 11:00 AM UTC · 629 words"},
		{name: "date only", createdAt: ts, words: 0, want: "Jul 17, 2026, 11:00 AM UTC"},
		{name: "words only", createdAt: time.Time{}, words: 12, want: "12 words"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatClipMeta(tt.createdAt, tt.words))
		})
	}
}
