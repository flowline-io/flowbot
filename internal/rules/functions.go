package rules

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/rules/components"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
	"github.com/flowline-io/flowbot/pkg/utils"
	ruleTypes "github.com/rulego/rulego/api/types"
)

func RegisterFunctions() {
	// register bot workflow rules
	for name, handler := range chatbot.List() {
		if !handler.IsReady() {
			continue
		}

		for _, botRuleSets := range handler.Rules() {
			switch v := botRuleSets.(type) {
			case []workflow.Rule:
				for i, botRule := range v {
					flog.Info("register rule function: %s/%s", name, botRule.Id)
					components.Functions.Register(fmt.Sprintf("%s/%s", name, botRule.Id), func(ctx ruleTypes.RuleContext, msg ruleTypes.RuleMsg) {
						botCtx := types.Context{
							Id:     msg.Id,
							AsUser: types.Uid(msg.Metadata.GetValue("uid")),
						}
						ctx.SetContext(ctx.GetContext())

						if msg.DataType != ruleTypes.JSON {
							ctx.TellFailure(msg, fmt.Errorf("invalid data type: %s", msg.DataType))
							return
						}

						var input types.KV
						err := sonic.Unmarshal(utils.StringToBytes(msg.GetData()), &input)
						if err != nil {
							ctx.TellFailure(msg, err)
							return
						}

						out, err := v[i].Run(botCtx, input)
						if err != nil {
							ctx.TellFailure(msg, err)
							return
						}

						msg.DataType = ruleTypes.JSON
						if out != nil {
							d, err := sonic.Marshal(out)
							if err != nil {
								ctx.TellFailure(msg, err)
								return
							}
							msg.SetData(utils.BytesToString(d))
						} else {
							msg.SetData("")
						}

						ctx.TellSuccess(msg)
					})
				}
			}
		}
	}

	// custom functions
	components.Functions.Register("sendMessage", func(ctx ruleTypes.RuleContext, msg ruleTypes.RuleMsg) {
		ctx.TellSuccess(msg)
	})
}
