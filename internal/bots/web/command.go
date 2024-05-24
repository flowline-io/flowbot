package web

import (
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/oneai"
)

var commandRules = []command.Rule{
	{
		Define: "summary [string]",
		Help:   `web page summary`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, _ := tokens[1].Value.String()
			// api key
			val, err := providers.GetConfig(oneai.ID, oneai.ApiKey)
			if err != nil {
				return types.TextMsg{Text: "error api config"}
			}

			api := oneai.NewOneAI(val.String())
			resp, err := api.Summarize(url)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error summarize"}
			}

			if len(resp.Output) != 2 || len(resp.Output[1].Contents) == 0 {
				return types.TextMsg{Text: "empty summarize"}
			}

			return types.TextMsg{Text: resp.Output[1].Contents[0].Utterance}
		},
	},
}
