package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	defaultPerTokenRPM        = 120
	defaultWritePerTokenRPM   = 20
	defaultLineagePerTokenRPM = 30
	defaultMaxArgumentBytes   = 65536
	rateWindow                = time.Minute
)

const rateScript = `
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local member = ARGV[3]
local retry = nil
for i = 1, #KEYS do
  local limit = tonumber(ARGV[3 + i])
  redis.call('ZREMRANGEBYSCORE', KEYS[i], '-inf', now - window)
  local count = redis.call('ZCARD', KEYS[i])
  if count >= limit then
    local oldest = redis.call('ZRANGE', KEYS[i], 0, 0, 'WITHSCORES')
    local wait = window
    if oldest[2] ~= nil then
      wait = tonumber(oldest[2]) + window - now
      if wait < 1 then
        wait = 1
      end
    end
    return {0, wait}
  end
end
for i = 1, #KEYS do
  redis.call('ZADD', KEYS[i], now, member)
  redis.call('PEXPIRE', KEYS[i], window)
end
return {1, 0}
`

type scriptEval interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd
}

var (
	rateRedis    scriptEval
	rateSequence atomic.Uint64
)

// SetRateRedis replaces the Redis client used for MCP tool rate limits.
// A nil client clears the override so the process client is used.
func SetRateRedis(c scriptEval) {
	rateRedis = c
}

func activeRateRedis() scriptEval {
	if rateRedis != nil {
		return rateRedis
	}
	if rdb.Client == nil {
		return nil
	}
	return rdb.Client
}

type rateBucket struct {
	name  string
	limit int
}

func maxArgumentBytes() int {
	if n := config.App.MCP.MaxArgumentBytes; n > 0 {
		return n
	}
	return defaultMaxArgumentBytes
}

func argumentByteCount(raw json.RawMessage) int {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	return len(raw)
}

func bucketsForTool(tokenID int64, spec ToolSpec, known bool) []rateBucket {
	cfg := config.App.MCP.RateLimit
	total := cfg.PerTokenRPM
	if total <= 0 {
		total = defaultPerTokenRPM
	}
	out := []rateBucket{{name: fmt.Sprintf("mcp:rl:%d:all", tokenID), limit: total}}
	if !known {
		return out
	}
	if isLineageTool(spec.Name) {
		limit := cfg.LineagePerTokenRPM
		if limit <= 0 {
			limit = defaultLineagePerTokenRPM
		}
		out = append(out, rateBucket{name: fmt.Sprintf("mcp:rl:%d:lineage", tokenID), limit: limit})
	}
	tier, _ := toolTier(spec)
	if tier != capability.TierRead {
		limit := cfg.WritePerTokenRPM
		if limit <= 0 {
			limit = defaultWritePerTokenRPM
		}
		out = append(out, rateBucket{name: fmt.Sprintf("mcp:rl:%d:write", tokenID), limit: limit})
	}
	return out
}

func isLineageTool(name string) bool {
	switch name {
	case "lineage.trace", "lineage.search", "flow.graph", "events.timeline":
		return true
	default:
		return false
	}
}

// allowToolCall records one tools/call against every applicable bucket.
// Redis errors fail closed. retryAfter is set when allowed is false.
func allowToolCall(ctx context.Context, buckets []rateBucket, now time.Time) (allowed bool, retryAfter time.Duration, err error) {
	client := activeRateRedis()
	if client == nil {
		return false, 0, types.Errorf(types.ErrUnavailable, "rate limiter unavailable")
	}
	keys := make([]string, len(buckets))
	args := make([]any, 0, 3+len(buckets))
	args = append(args, now.UnixMilli(), rateWindow.Milliseconds(), rateMember(now))
	for i, bucket := range buckets {
		keys[i] = bucket.name
		args = append(args, bucket.limit)
	}
	raw, err := client.Eval(ctx, rateScript, keys, args...).Result()
	if err != nil {
		return false, 0, types.Errorf(types.ErrUnavailable, "rate limiter unavailable")
	}
	vals, ok := raw.([]any)
	if !ok || len(vals) < 2 {
		return false, 0, types.Errorf(types.ErrUnavailable, "rate limiter unavailable")
	}
	allowedN, ok := asInt64(vals[0])
	if !ok {
		return false, 0, types.Errorf(types.ErrUnavailable, "rate limiter unavailable")
	}
	if allowedN == 1 {
		return true, 0, nil
	}
	retryMS, ok := asInt64(vals[1])
	if !ok || retryMS < 1 {
		retryMS = 1
	}
	return false, time.Duration(retryMS) * time.Millisecond, nil
}

func rateMember(now time.Time) string {
	return fmt.Sprintf("%d:%d", now.UnixNano(), rateSequence.Add(1))
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}
