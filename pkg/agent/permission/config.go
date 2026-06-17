package permission

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/bytedance/sonic"
)

// Config maps permission keys to rule sets (OpenCode-compatible shape).
type Config map[string]RuleSet

// RuleSet is either one global action or an ordered list of pattern rules.
type RuleSet struct {
	Default  Action
	Patterns []PatternRule
}

// PatternRule binds one glob pattern to an action; last match wins during evaluation.
type PatternRule struct {
	Pattern string
	Action  Action
}

// ParseConfig unmarshals permission JSON (string or pattern map per key).
func ParseConfig(raw []byte) (Config, error) {
	if len(raw) == 0 {
		return Config{}, nil
	}
	var root map[string]json.RawMessage
	if err := sonic.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse permission config: %w", err)
	}
	out := make(Config, len(root))
	for key, value := range root {
		rs, err := parseRuleSet([]byte(value))
		if err != nil {
			return nil, fmt.Errorf("permission key %q: %w", key, err)
		}
		out[key] = rs
	}
	return out, nil
}

func parseRuleSet(raw []byte) (RuleSet, error) {
	var simple string
	if err := sonic.Unmarshal(raw, &simple); err == nil {
		action, ok := ParseAction(simple)
		if !ok {
			return RuleSet{}, fmt.Errorf("invalid action %q", simple)
		}
		return RuleSet{Default: action}, nil
	}
	var patterns map[string]string
	if err := sonic.Unmarshal(raw, &patterns); err != nil {
		return RuleSet{}, fmt.Errorf("invalid rule value")
	}
	keys := make([]string, 0, len(patterns))
	for pattern := range patterns {
		keys = append(keys, pattern)
	}
	slices.Sort(keys)
	rules := make([]PatternRule, 0, len(keys))
	for _, pattern := range keys {
		action, ok := ParseAction(patterns[pattern])
		if !ok {
			return RuleSet{}, fmt.Errorf("invalid action %q for pattern %q", patterns[pattern], pattern)
		}
		rules = append(rules, PatternRule{Pattern: pattern, Action: action})
	}
	return RuleSet{Patterns: rules}, nil
}

// Merge overlays user rules on top of defaults; user keys replace defaults entirely.
func Merge(base, overlay Config) Config {
	if len(base) == 0 && len(overlay) == 0 {
		return Config{}
	}
	out := make(Config, len(base)+len(overlay))
	maps.Copy(out, base)
	maps.Copy(out, overlay)
	return out
}

// MarshalJSON serializes the config for API responses.
func (c Config) MarshalJSON() ([]byte, error) {
	root := make(map[string]any, len(c))
	for key, rs := range c {
		if len(rs.Patterns) == 0 {
			root[key] = string(rs.Default)
			continue
		}
		patterns := make(map[string]string, len(rs.Patterns))
		for _, rule := range rs.Patterns {
			patterns[rule.Pattern] = string(rule.Action)
		}
		root[key] = patterns
	}
	return sonic.Marshal(root)
}
