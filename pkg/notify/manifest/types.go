// Package manifest holds shared notification template and rule type definitions.
// It is a leaf package so rules and template engines can import it without cycles.
package manifest

// Template defines a notification message template with optional per-channel overrides.
type Template struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	DefaultFormat   string     `json:"default_format"`
	DefaultTemplate string     `json:"default_template"`
	Overrides       []Override `json:"overrides"`
}

// Override defines a channel-specific template override.
type Override struct {
	Channel  string `json:"channel"`
	Format   string `json:"format"`
	Template string `json:"template"`
}

// RuleAction defines the action to take when a rule matches.
type RuleAction string

// Rule action constants.
const (
	RuleActionThrottle  RuleAction = "throttle"
	RuleActionAggregate RuleAction = "aggregate"
	RuleActionMute      RuleAction = "mute"
	RuleActionDrop      RuleAction = "drop"
)

// RuleMatch defines the event and channel matching criteria.
type RuleMatch struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

// RuleParams holds action-specific parameters.
type RuleParams struct {
	Window      string `json:"window"`
	Limit       int    `json:"limit"`
	DigestTplID string `json:"digest_template_id"`
	DelayedSend bool   `json:"delayed_send"`
}

// Rule defines a notification filtering or aggregation rule.
type Rule struct {
	ID        string     `json:"id"`
	Action    RuleAction `json:"action"`
	Match     RuleMatch  `json:"match"`
	Condition string     `json:"condition"`
	Priority  int        `json:"priority"`
	Params    RuleParams `json:"params"`
}
