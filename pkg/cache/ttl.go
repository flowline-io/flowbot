package cache

import "time"

// TTL represents a cache time-to-live duration.
// It wraps time.Duration to provide named constants
// for common cache expiry policies.
type TTL time.Duration

// Predefined TTL constants for standardized cache expiry policies.
const (
	// TTLNone disables expiry — items persist until explicitly deleted or evicted.
	TTLNone TTL = 0
	// TTLMinute is a 1-minute lifetime, suitable for transient locks or status flags.
	TTLMinute TTL = TTL(time.Minute)
	// TTLShort is a 2-minute lifetime, used for heartbeat and liveness signals.
	TTLShort TTL = TTL(2 * time.Minute)
	// TTLMedium is a 10-minute lifetime, used for temporary state with moderate freshness requirements.
	TTLMedium TTL = TTL(10 * time.Minute)
	// TTLLong is a 1-hour lifetime, used for cached data that can tolerate slight staleness.
	TTLLong TTL = TTL(1 * time.Hour)
	// TTLSession is a 24-hour lifetime, used for chat sessions and user presence.
	TTLSession TTL = TTL(24 * time.Hour)
	// TTLDay is a 24-hour lifetime, used for daily deduplication windows.
	TTLDay TTL = TTL(24 * time.Hour)
	// TTLWeek is a 7-day lifetime, used for medium-term deduplication.
	TTLWeek TTL = TTL(7 * 24 * time.Hour)
	// TTLMonth is a 30-day lifetime, used for long-term deduplication to prevent unbounded key growth.
	TTLMonth TTL = TTL(30 * 24 * time.Hour)
)

// Duration returns the TTL as a standard time.Duration for use with cache backends.
func (t TTL) Duration() time.Duration {
	return time.Duration(t)
}
