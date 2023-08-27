package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/condition"
	"github.com/sysatom/flowbot/internal/types"
)

var conditionRules = []condition.Rule{
	{
		Condition: "RepoMsg",
		Handler: func(ctx types.Context, forwarded types.MsgPayload) types.MsgPayload {
			repo, _ := forwarded.(types.RepoMsg)
			return types.TextMsg{Text: *repo.FullName}
		},
	},
}
