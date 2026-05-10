package bookmark

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventRules(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 3 event rules",
			test: func(t *testing.T) {
				assert.Len(t, eventRules, 3)
			},
		},
		{
			name: "should contain expected event IDs",
			test: func(t *testing.T) {
				ids := make(map[string]bool)
				for _, r := range eventRules {
					ids[r.Id] = true
				}

				assert.True(t, ids[types.BookmarkArchiveBotEventID])
				assert.True(t, ids[types.BookmarkCreateBotEventID])
				assert.True(t, ids[types.ArchiveBoxAddBotEventID])
			},
		},
		{
			name: "all event rules should have non-nil handlers",
			test: func(t *testing.T) {
				for _, r := range eventRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
