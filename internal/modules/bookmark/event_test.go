package bookmark

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestEventRules_Count(t *testing.T) {
	assert.Len(t, eventRules, 3)
}

func TestEventRules_IDs(t *testing.T) {
	ids := make(map[string]bool)
	for _, r := range eventRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[types.BookmarkArchiveBotEventID])
	assert.True(t, ids[types.BookmarkCreateBotEventID])
	assert.True(t, ids[types.ArchiveBoxAddBotEventID])
}

func TestEventRules_Handlers(t *testing.T) {
	for _, r := range eventRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
	}
}
