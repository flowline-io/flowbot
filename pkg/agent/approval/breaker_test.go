package approval_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/stretchr/testify/assert"
)

func TestBreakerTripsAndLatches(t *testing.T) {
	b := approval.NewBreaker(3)
	assert.False(t, b.RecordDenial())
	assert.False(t, b.RecordDenial())
	assert.True(t, b.RecordDenial())
	assert.True(t, b.Tripped())
	assert.True(t, b.RecordDenial())
	b.Reset()
	assert.True(t, b.Tripped())
	assert.Equal(t, 3, b.Count())
}

func TestBreakerResetClearsCount(t *testing.T) {
	b := approval.NewBreaker(3)
	assert.False(t, b.RecordDenial())
	b.Reset()
	assert.Equal(t, 0, b.Count())
	assert.False(t, b.Tripped())
}
