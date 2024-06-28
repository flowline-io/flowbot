package locker

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"github.com/flowline-io/flowbot/pkg/cache"
)

type Locker struct {
	lock *redislock.Client
}

func NewLocker() *Locker {
	return &Locker{lock: redislock.New(cache.DB)}
}

func (l *Locker) Acquire(ctx context.Context, key string, ttl time.Duration) (*redislock.Lock, error) {
	return l.lock.Obtain(ctx, key, ttl, nil)
}
