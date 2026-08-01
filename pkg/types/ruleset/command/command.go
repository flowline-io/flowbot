// Package command implements the command ruleset type.
package command

import (
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Define  string
	Help    string
	Handler func(types.Context, []*parser.Token) types.MsgPayload
}

func (r Rule) ID() string {
	return r.Define
}

func (Rule) TYPE() types.RulesetType {
	return types.CommandRule
}

// FormatHelpLine formats a single command help entry for MarkdownMsg.
func (r Rule) FormatHelpLine() string {
	return "`/" + r.Define + "` — " + r.Help
}

type Ruleset []Rule

func (r Ruleset) Help(in string) (types.MsgPayload, error) {
	lower := strings.ToLower(in)
	if lower != "help" && lower != "h" {
		return nil, nil
	}
	if len(r) == 0 {
		return nil, nil
	}

	rules := slices.Clone(r)
	slices.SortFunc(rules, func(a, b Rule) int {
		return strings.Compare(a.Define, b.Define)
	})

	lines := make([]string, 0, len(rules))
	for _, rule := range rules {
		lines = append(lines, rule.FormatHelpLine())
	}

	return types.MarkdownMsg{
		Title: "Help",
		Raw:   strings.Join(lines, "\n"),
	}, nil
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
