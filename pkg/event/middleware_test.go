package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry_Middleware(t *testing.T) {
	retry := Retry{
		MaxRetries:          3,
		InitialInterval:     1 * time.Second,
		MaxInterval:         30 * time.Second,
		Multiplier:          2.0,
		MaxElapsedTime:      2 * time.Minute,
		RandomizationFactor: 0.5,
	}

	assert.Equal(t, 3, retry.MaxRetries)
	assert.Equal(t, 1*time.Second, retry.InitialInterval)
	assert.Equal(t, 30*time.Second, retry.MaxInterval)
	assert.Equal(t, 2.0, retry.Multiplier)
	assert.Equal(t, 2*time.Minute, retry.MaxElapsedTime)
	assert.Equal(t, 0.5, retry.RandomizationFactor)
}
