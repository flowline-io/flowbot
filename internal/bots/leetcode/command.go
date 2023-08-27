package leetcode

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/cache"
	"github.com/sysatom/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "info",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
	{
		Define: "pick [string]",
		Help:   `pick one [easy|medium|hard]`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			level, _ := tokens[1].Value.String()
			data, err := cache.DB.SRandMember(context.Background(), fmt.Sprintf("leetcode:problems:%s", level)).Bytes()
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			var p Problem
			err = json.Unmarshal(data, &p)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.QuestionMsg{
				Id:         p.Stat.FrontendQuestionID,
				Title:      p.Stat.QuestionTitle,
				Slug:       p.Stat.QuestionTitleSlug,
				Difficulty: p.Difficulty.Level,
				Source:     "leetcode",
			}
		},
	},
}
