package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// RateLimiter handles rate limiting for flows and nodes
type RateLimiter struct {
	store store.Adapter
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(storeAdapter store.Adapter) *RateLimiter {
	return &RateLimiter{
		store: storeAdapter,
	}
}

// CheckRateLimit checks if a flow or node can be executed based on rate limits
func (r *RateLimiter) CheckRateLimit(ctx context.Context, flowID *int64, nodeID string) (bool, error) {
	limits, err := r.store.GetRateLimits(flowID, nodeID)
	if err != nil {
		return false, fmt.Errorf("failed to get rate limits: %w", err)
	}

	if len(limits) == 0 {
		return true, nil
	}

	// Check each rate limit
	for _, limit := range limits {
		allowed, err := r.checkLimit(ctx, limit)
		if err != nil {
			flog.Error(fmt.Errorf("failed to check rate limit: %w", err))
			continue
		}
		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

// checkLimit checks a single rate limit
func (r *RateLimiter) checkLimit(ctx context.Context, limit *model.RateLimit) (bool, error) {
	// Calculate window duration
	windowDuration := time.Duration(limit.WindowSize)
	switch limit.WindowUnit {
	case "second":
		windowDuration *= time.Second
	case "minute":
		windowDuration *= time.Minute
	case "hour":
		windowDuration *= time.Hour
	case "day":
		windowDuration *= 24 * time.Hour
	default:
		windowDuration *= time.Second
	}

	// Get execution count in window
	count, err := r.getExecutionCount(ctx, limit, windowDuration)
	if err != nil {
		return false, err
	}

	return count < limit.LimitValue, nil
}

// getExecutionCount gets the execution count within the time window
func (r *RateLimiter) getExecutionCount(ctx context.Context, limit *model.RateLimit, windowDuration time.Duration) (int, error) {
	// This is a simplified implementation
	// In production, you might want to use Redis or a more efficient counting mechanism
	windowStart := time.Now().Add(-windowDuration)

	var flowID int64
	if limit.FlowID != nil {
		flowID = *limit.FlowID
	}

	// Get executions and filter by time window
	// Note: This could be optimized by adding a time-based query to the store adapter
	executions, err := r.store.GetExecutions(flowID, 1000)
	if err != nil {
		return 0, err
	}

	// Count executions within the time window
	// Since executions are typically ordered by creation time (newest first),
	// we can break early if we find an execution outside the window
	count := 0
	for _, exec := range executions {
		if exec.CreatedAt.After(windowStart) || exec.CreatedAt.Equal(windowStart) {
			count++
		} else {
			// If executions are ordered by time descending, we can break early
			// Otherwise, continue checking all records
			break
		}
	}

	return count, nil
}
