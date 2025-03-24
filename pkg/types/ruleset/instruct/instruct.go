package instruct

import "github.com/flowline-io/flowbot/pkg/types"

type Rule struct {
	Id   string
	Args []string
}

func (r Rule) ID() string {
	return r.Id
}

func (r Rule) TYPE() types.RulesetType {
	return types.InstructRule
}

type Ruleset []Rule
