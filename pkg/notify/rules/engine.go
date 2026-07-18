// Package rules provides the notification rule engine for throttling, aggregation, and mute/DND.
package rules

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

// Engine evaluates notification rules to determine whether a message should be sent,
// throttled, aggregated, or dropped.
type Engine struct {
	mu    sync.RWMutex
	rules []manifest.Rule
	store *cache.RedisStore
}

// globalEngine is the singleton rule engine.
var globalEngine struct {
	mu     sync.RWMutex
	engine *Engine
}

// Init initializes the global rule engine with the given rules and a RedisStore.
func Init(store *cache.RedisStore, rules []manifest.Rule) error {
	engine := New(store)
	if err := engine.LoadConfig(rules); err != nil {
		return err
	}

	globalEngine.mu.Lock()
	globalEngine.engine = engine
	globalEngine.mu.Unlock()

	flog.Info("notify rules engine: loaded %d rules", len(rules))
	return nil
}

// GetEngine returns the global rule engine.
func GetEngine() *Engine {
	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()
	return globalEngine.engine
}

// New creates a new rule Engine.
func New(store *cache.RedisStore) *Engine {
	return &Engine{
		store: store,
	}
}

// LoadConfig loads and sorts rules from configuration.
func (e *Engine) LoadConfig(rules []manifest.Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// sort by priority descending (higher priority first)
	sorted := make([]manifest.Rule, len(rules))
	copy(sorted, rules)
	slices.SortFunc(sorted, func(a, b manifest.Rule) int {
		return b.Priority - a.Priority
	})

	e.rules = sorted
	return nil
}

// Reload refreshes the rule list from the database.
// Called after rule CRUD operations to enable hot-reload without restart.
func (e *Engine) Reload(ctx context.Context, loader func(context.Context) ([]manifest.Rule, error)) error {
	rules, err := loader(ctx)
	if err != nil {
		return err
	}
	return e.LoadConfig(rules)
}

// EvalResult represents the outcome of rule evaluation.
type EvalResult struct {
	Action manifest.RuleAction
	RuleID string
	Window string
	Limit  int
	Muted  bool
}

// Evaluate checks all rules against an event type and channel, returning the first matching action.
func (e *Engine) Evaluate(_ context.Context, eventType, channel string) *EvalResult {
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
	if before, ok := strings.CutSuffix(pattern, ".*"); ok {
		prefix := before
		return strings.HasPrefix(value, prefix+".")
	}
	// prefix match: "*.created" matches "bookmark.created"
	if after, ok := strings.CutPrefix(pattern, "*."); ok {
		suffix := after
		return strings.HasSuffix(value, "."+suffix)
	}
	return false
}

// evalCondition evaluates a simple time-based condition expression.
// Supported: time.hour >= N, time.hour < N, connected by || and &&.
func evalCondition(condition string) bool {
	// For simplicity, split by || and evaluate each part
	parts := strings.SplitSeq(condition, "||")
	for part := range parts {
		part = strings.TrimSpace(part)
		if evalSimpleCondition(part) {
			return true
		}
	}
	return false
}

func evalSimpleCondition(condition string) bool {
	// handle && within a part
	andParts := strings.SplitSeq(condition, "&&")
	for part := range andParts {
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

// ValidateCondition checks whether a condition expression string is syntactically valid.
// It uses the same parsing logic as evalCondition but without evaluating time values.
func ValidateCondition(condition string) error {
	if condition == "" {
		return nil
	}
	// validate each || part
	parts := strings.SplitSeq(condition, "||")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("rules: empty expression after ||")
		}
		andParts := strings.SplitSeq(part, "&&")
		for ap := range andParts {
			ap = strings.TrimSpace(ap)
			if ap == "" {
				return fmt.Errorf("rules: empty expression after &&")
			}
			if err := validateTimeExpression(ap); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTimeExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "time.hour ") {
		return fmt.Errorf("rules: expected 'time.hour <op> N', got %q", expr)
	}
	for _, op := range []string{">=", "<=", "==", ">", "<"} {
		if strings.Contains(expr, " "+op+" ") || strings.HasPrefix(expr[len("time.hour "):], op+" ") {
			return nil
		}
	}
	return fmt.Errorf("rules: unknown operator in %q", expr)
}
