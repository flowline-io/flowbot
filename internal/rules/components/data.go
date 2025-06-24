package components

import (
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	ruleTypes "github.com/rulego/rulego/api/types"
)

type DataNode struct {
	data types.KV
}

func (n *DataNode) Type() string {
	return "flowbot/data"
}

func (n *DataNode) New() ruleTypes.Node {
	return &DataNode{}
}

func (n *DataNode) Init(_ ruleTypes.Config, configuration ruleTypes.Configuration) error {
	n.data = types.KV(configuration)
	return nil
}

func (n *DataNode) OnMsg(ctx ruleTypes.RuleContext, msg ruleTypes.RuleMsg) {
	b, err := sonic.Marshal(n.data)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	msg.DataType = ruleTypes.JSON
	msg.SetData(utils.BytesToString(b))
	ctx.TellSuccess(msg)
}

func (n *DataNode) Destroy() {}
