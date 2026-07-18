package notify

import "github.com/flowline-io/flowbot/pkg/notify/manifest"

// Type aliases re-export manifest types so callers can use notify.Template / notify.Rule
// without importing the leaf package. Subpackages (rules, template) import manifest
// directly to avoid an import cycle with this package.

// Template is a notification message template with optional per-channel overrides.
type Template = manifest.Template

// Override is a channel-specific template override.
type Override = manifest.Override

// RuleAction is the action to take when a rule matches.
type RuleAction = manifest.RuleAction

// Rule action constants.
const (
	RuleActionThrottle  = manifest.RuleActionThrottle
	RuleActionAggregate = manifest.RuleActionAggregate
	RuleActionMute      = manifest.RuleActionMute
	RuleActionDrop      = manifest.RuleActionDrop
)

// RuleMatch is the event and channel matching criteria.
type RuleMatch = manifest.RuleMatch

// RuleParams holds action-specific parameters.
type RuleParams = manifest.RuleParams

// Rule is a notification filtering or aggregation rule.
type Rule = manifest.Rule
