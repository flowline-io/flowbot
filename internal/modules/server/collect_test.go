package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectConstants(t *testing.T) {
	assert.Equal(t, "stats_collect", StatsCollectID)
}

func TestCollectRules_Count(t *testing.T) {
	assert.Len(t, collectRules, 1)
}

func TestCollectRules_ID(t *testing.T) {
	assert.Equal(t, StatsCollectID, collectRules[0].Id)
}

func TestCollectRules_Help(t *testing.T) {
	assert.Equal(t, "upload server status", collectRules[0].Help)
}

func TestCollectRules_Args(t *testing.T) {
	assert.Equal(t, []string{"cpu", "memory", "info"}, collectRules[0].Args)
}

func TestCollectRules_Handler(t *testing.T) {
	assert.NotNil(t, collectRules[0].Handler)
}
