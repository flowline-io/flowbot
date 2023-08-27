package pipeline

import (
	"errors"
	"fmt"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/parser"
	"strings"
)

type Rule struct {
	Id      string
	Version int
	Help    string
	Trigger types.Trigger
	Step    []types.Stage
}

type Ruleset []Rule

func (r Ruleset) Help(in string) (types.MsgPayload, error) {
	if strings.ToLower(in) == "help" {
		m := make(map[string]interface{})
		for _, rule := range r {
			switch rule.Trigger.Type {
			case types.TriggerCommandType:
				m[fmt.Sprintf("~%s", rule.Trigger.Define)] = rule.Help
			}
		}

		return types.InfoMsg{
			Title: "Pipeline",
			Model: m,
		}, nil
	}
	return nil, nil
}

func (r Ruleset) TriggerPipeline(_ types.Context, triggerType types.TriggerType, in string) (Rule, error) {
	switch triggerType {
	case types.TriggerCommandType:
		for _, rule := range r {
			tokens, err := parser.ParseString(in)
			if err != nil {
				return Rule{}, err
			}
			check, err := parser.SyntaxCheck(rule.Trigger.Define, tokens)
			if err != nil {
				return Rule{}, err
			}
			if !check {
				continue
			}
			return rule, nil
		}
	}
	return Rule{}, errors.New("not match trigger")
}
