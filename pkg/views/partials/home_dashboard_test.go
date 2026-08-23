package partials

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeQuickLinks(t *testing.T) {
	t.Parallel()
	links := HomeQuickLinks(context.Background())
	require.Len(t, links, 4)

	tests := []struct {
		idx    int
		title  string
		href   string
		testID string
	}{
		{0, "Knowledge", "/service/web/agent-knowledge", "home-link-knowledge"},
		{1, "Skills", "/service/web/agent-skills", "home-link-skills"},
		{2, "Notifications", "/service/web/notifications", "home-link-notifications"},
		{3, "Health", "/service/web/healthz", "home-link-healthz"},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.title, links[tt.idx].Title)
			assert.Equal(t, tt.href, links[tt.idx].Href)
			assert.Equal(t, tt.testID, links[tt.idx].TestID)
			assert.NotEmpty(t, links[tt.idx].Detail)
		})
	}
}
