package command

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Define  string
	Help    string
	Handler func(types.Context, []*parser.Token) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) Help(in string) (types.MsgPayload, error) {
	if strings.ToLower(in) == "help" || strings.ToLower(in) == "h" {
		m := make(types.KV)
		for _, rule := range r {
			m[fmt.Sprintf("/%s", rule.Define)] = rule.Help
		}

		return types.InfoMsg{
			Title: "Help",
			Model: m,
		}, nil
	}
	return nil, nil
}

func (r Ruleset) ProcessCommand(ctx types.Context, in string) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		tokens, err := parser.ParseString(in)
		if err != nil {
			return nil, err
		}
		check, err := parser.SyntaxCheck(rule.Define, tokens)
		if err != nil {
			return nil, err
		}
		if !check {
			continue
		}
		result = rule.Handler(ctx, tokens)
	}
	return result, nil
}
