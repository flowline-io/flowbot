package anki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectRules_Count(t *testing.T) {
	assert.Len(t, collectRules, 2)
}

func TestCollectRules_IDs(t *testing.T) {
	ids := make(map[string]bool)
	for _, r := range collectRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[StatsCollectID])
	assert.True(t, ids[ReviewCollectID])
}

func TestCollectRules_Constants(t *testing.T) {
	assert.Equal(t, "stats_collect", StatsCollectID)
	assert.Equal(t, "review_collect", ReviewCollectID)
}

func TestCollectRules_Args(t *testing.T) {
	for _, r := range collectRules {
		switch r.Id {
		case StatsCollectID:
			assert.Equal(t, []string{"html"}, r.Args)
		case ReviewCollectID:
			assert.Equal(t, []string{"num"}, r.Args)
		}
	}
}

func TestCollectRules_Handlers(t *testing.T) {
	for _, r := range collectRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
	}
}
