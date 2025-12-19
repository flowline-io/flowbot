package trigger

import (
	"github.com/flowline-io/flowbot/pkg/flows"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Mode string

type Abstraction string

const (
	ModeWebhook Mode = "webhook"
	ModePoll    Mode = "poll"
	ModeManual  Mode = "manual"

	AbstractionObject     Abstraction = "object"
	AbstractionTransition Abstraction = "transition"
)

type PollResult struct {
	Events []types.KV `json:"events"`
	State  types.KV   `json:"state"`
}

// Rule defines a Flow trigger.
//
// - Extract is executed inside the flow engine when a trigger event is processed.
// - Poll is executed by the flow poller when Mode is ModePoll.
//
// Ingredients describes available variables the trigger can produce.
// The engine does not enforce it, but the Flow Editor can display it.
type Rule struct {
	Id          string
	Title       string
	Description string

	Mode        Mode
	Abstraction Abstraction

	Ingredients []flows.Ingredient

	// Config validates node parameters.
	Config func(params types.KV) error

	// Extract converts the incoming payload into ingredients.
	Extract func(ctx types.Context, params types.KV, payload types.KV) (types.KV, error)

	// Poll periodically fetches events and maintains state.
	Poll func(ctx types.Context, params types.KV, state types.KV) (PollResult, error)
}

func (r Rule) ID() string {
	return r.Id
}

func (r Rule) TYPE() types.RulesetType {
	return types.TriggerRule
}

type Ruleset []Rule
