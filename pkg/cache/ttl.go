package cache

import "time"

type TTL time.Duration

const (
	TTLNone    TTL = 0
	TTLMinute  TTL = TTL(time.Minute)
	TTLShort   TTL = TTL(2 * time.Minute)
	TTLMedium  TTL = TTL(10 * time.Minute)
	TTLLong    TTL = TTL(1 * time.Hour)
	TTLSession TTL = TTL(24 * time.Hour)
	TTLDay     TTL = TTL(24 * time.Hour)
	TTLWeek    TTL = TTL(7 * 24 * time.Hour)
	TTLMonth   TTL = TTL(30 * 24 * time.Hour)
)

func (t TTL) Duration() time.Duration {
	return time.Duration(t)
}
