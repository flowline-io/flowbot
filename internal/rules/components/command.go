package components

import (
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	ruleTypes "github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/utils/maps"
)

type CommandNodeConfiguration struct {
	Command string `json:"command" yaml:"command"`
}

type CommandNode struct {
	Config CommandNodeConfiguration
}

func (n *CommandNode) Type() string {
	return "flowbot/command"
}

func (n *CommandNode) New() ruleTypes.Node {
	return &CommandNode{}
}

func (n *CommandNode) Init(ruleConfig ruleTypes.Config, configuration ruleTypes.Configuration) error {
	err := maps.Map2Struct(configuration, &n.Config)
	if err != nil {
		return err
	}
	return nil
}

func (n *CommandNode) OnMsg(ctx ruleTypes.RuleContext, msg ruleTypes.RuleMsg) {
	botCtx := types.Context{
		Id:     msg.Id,
		AsUser: types.Uid(msg.Metadata.GetValue("uid")),
	}
	ctx.SetContext(ctx.GetContext())

	for _, handle := range chatbot.List() {
		if !handle.IsReady() {
			continue
		}
		payload, err := handle.Command(botCtx, n.Config.Command)
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}

		if payload != nil {
			d, err := sonic.Marshal(payload)
			if err != nil {
				ctx.TellFailure(msg, err)
				return
			}

			stats.BotRunTotalCounter(stats.CommandRuleset).Inc()
			msg.Data = utils.BytesToString(d)
			ctx.TellSuccess(msg)
			return
		}
	}
}

func (n *CommandNode) Destroy() {}
