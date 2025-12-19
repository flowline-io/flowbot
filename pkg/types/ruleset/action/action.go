package action

import (
	"github.com/flowline-io/flowbot/pkg/flows"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Rule defines a Flow action.
//
// Params are the node parameters after template substitution.
// Variables contains the accumulated ingredients and intermediate variables.
type Rule struct {
	Id          string
	Title       string
	Description string

	Inputs []flows.ParamSpec

	// Validate validates params. If nil, flows.ValidateParams(Inputs) is used.
	Validate func(params types.KV) error

	// Run executes the action.
	Run func(ctx types.Context, params types.KV, variables types.KV) (types.KV, error)
}

func (r Rule) ID() string {
	return r.Id
}

func (r Rule) TYPE() types.RulesetType {
	return types.ActionRule
}

type Ruleset []Rule
