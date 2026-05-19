package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTTLDuration verifies that each TTL constant produces the expected time.Duration
// when calling the Duration() method.
func TestTTLDuration(t *testing.T) {
	tests := []struct {
		name    string
		ttl     TTL
		wantDur time.Duration
	}{
		{
			name:    "TTLNone is zero",
			ttl:     TTLNone,
			wantDur: 0,
		},
		{
			name:    "TTLMinute is one minute",
			ttl:     TTLMinute,
			wantDur: time.Minute,
		},
		{
			name:    "TTLShort is two minutes",
			ttl:     TTLShort,
			wantDur: 2 * time.Minute,
		},
		{
			name:    "TTLMedium is ten minutes",
			ttl:     TTLMedium,
			wantDur: 10 * time.Minute,
		},
		{
			name:    "TTLLong is one hour",
			ttl:     TTLLong,
			wantDur: time.Hour,
		},
		{
			name:    "TTLSession is 24 hours",
			ttl:     TTLSession,
			wantDur: 24 * time.Hour,
		},
		{
			name:    "TTLDay is 24 hours",
			ttl:     TTLDay,
			wantDur: 24 * time.Hour,
		},
		{
			name:    "TTLWeek is 7 days",
			ttl:     TTLWeek,
			wantDur: 7 * 24 * time.Hour,
		},
		{
			name:    "TTLMonth is 30 days",
			ttl:     TTLMonth,
			wantDur: 30 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantDur, tt.ttl.Duration())
		})
	}
}
