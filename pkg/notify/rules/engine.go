// Package rules provides the notification rule engine for throttling, aggregation, and mute/DND.
package rules

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Engine evaluates notification rules to determine whether a message should be sent,
// throttled, aggregated, or dropped.
type Engine struct {
	mu    sync.RWMutex
	rules []config.NotifyRule
	redis *redis.Client
}

// globalEngine is the singleton rule engine.
var globalEngine struct {
	mu     sync.RWMutex
	engine *Engine
}

// Init initializes the global rule engine with configuration and a Redis client.
func Init(redisClient *redis.Client) error {
	engine := New(redisClient)
	if err := engine.LoadConfig(config.App.Notify.Rules); err != nil {
		return err
	}

	globalEngine.mu.Lock()
	globalEngine.engine = engine
	globalEngine.mu.Unlock()

	flog.Info("notify rules engine: loaded %d rules", len(config.App.Notify.Rules))
	return nil
}

// GetEngine returns the global rule engine.
func GetEngine() *Engine {
	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()
	return globalEngine.engine
}

// New creates a new rule Engine.
func New(redisClient *redis.Client) *Engine {
	return &Engine{
		redis: redisClient,
	}
}

// LoadConfig loads and sorts rules from configuration.
func (e *Engine) LoadConfig(rules []config.NotifyRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// sort by priority descending (higher priority first)
	sorted := make([]config.NotifyRule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	e.rules = sorted
	return nil
}

// EvalResult represents the outcome of rule evaluation.
type EvalResult struct {
	Action config.NotifyRuleAction
	RuleID string
	Window string
	Limit  int
	Muted  bool
}

// Evaluate checks all rules against an event type and channel, returning the first matching action.
func (e *Engine) Evaluate(ctx context.Context, eventType, channel string) *EvalResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		// match event pattern
		if !matchPattern(rule.Match.Event, eventType) {
			continue
		}
		// match channel pattern
		if !matchPattern(rule.Match.Channel, channel) {
			continue
		}

		// check condition (time-based mute, etc.)
		if rule.Condition != "" {
			if evalCondition(rule.Condition) {
				return &EvalResult{
					Action: rule.Action,
					RuleID: rule.ID,
					Window: rule.Params.Window,
					Limit:  rule.Params.Limit,
					Muted:  true,
				}
			}
			continue
		}

		return &EvalResult{
			Action: rule.Action,
			RuleID: rule.ID,
			Window: rule.Params.Window,
			Limit:  rule.Params.Limit,
		}
	}

	return nil
}

// matchPattern checks if a value matches a glob-like pattern.
// Supports "*" for match-all and simple prefix/suffix matching.
func matchPattern(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == value {
		return true
	}
	// suffix match: "infra.*" matches "infra.host.down"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(value, prefix+".")
	}
	// prefix match: "*.created" matches "bookmark.created"
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(value, "."+suffix)
	}
	return false
}

// evalCondition evaluates a simple time-based condition expression.
// Supported: time.hour >= N, time.hour < N, connected by || and &&.
func evalCondition(condition string) bool {
	// For simplicity, split by || and evaluate each part
	parts := strings.Split(condition, "||")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if evalSimpleCondition(part) {
			return true
		}
	}
	return false
}

func evalSimpleCondition(condition string) bool {
	// handle && within a part
	andParts := strings.Split(condition, "&&")
	for _, part := range andParts {
		part = strings.TrimSpace(part)
		if !evalTimeCondition(part) {
			return false
		}
	}
	return true
}

func evalTimeCondition(condition string) bool {
	condition = strings.TrimSpace(condition)
	now := currentHour()

	// parse: time.hour >= N or time.hour < N
	if strings.Contains(condition, ">=") {
		parts := strings.SplitN(condition, ">=", 2)
		if strings.TrimSpace(parts[0]) == "time.hour" {
			hour := parseHour(strings.TrimSpace(parts[1]))
			return now >= hour
		}
	}
	if strings.Contains(condition, "<=") {
		parts := strings.SplitN(condition, "<=", 2)
		if strings.TrimSpace(parts[0]) == "time.hour" {
			hour := parseHour(strings.TrimSpace(parts[1]))
			return now <= hour
		}
	}
	if strings.Contains(condition, "<") {
		parts := strings.SplitN(condition, "<", 2)
		if strings.TrimSpace(parts[0]) == "time.hour" {
			hour := parseHour(strings.TrimSpace(parts[1]))
			return now < hour
		}
	}
	if strings.Contains(condition, ">") {
		parts := strings.SplitN(condition, ">", 2)
		if strings.TrimSpace(parts[0]) == "time.hour" {
			hour := parseHour(strings.TrimSpace(parts[1]))
			return now > hour
		}
	}
	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		if strings.TrimSpace(parts[0]) == "time.hour" {
			hour := parseHour(strings.TrimSpace(parts[1]))
			return now == hour
		}
	}
	return false
}
