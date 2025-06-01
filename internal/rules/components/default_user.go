package components

import (
	"github.com/flowline-io/flowbot/internal/store"
	ruleTypes "github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/utils/maps"
)

type DefaultUserNodeConfiguration struct {
	Uid string `json:"uid" yaml:"uid"`
}

type DefaultUserNode struct {
	Config DefaultUserNodeConfiguration
}

func (n *DefaultUserNode) Type() string {
	return "flowbot/default_user"
}

func (n *DefaultUserNode) New() ruleTypes.Node {
	return &DefaultUserNode{}
}

func (n *DefaultUserNode) Init(_ ruleTypes.Config, configuration ruleTypes.Configuration) error {
	err := maps.Map2Struct(configuration, &n.Config)
	if err != nil {
		return err
	}
	return nil
}

func (n *DefaultUserNode) OnMsg(ctx ruleTypes.RuleContext, msg ruleTypes.RuleMsg) {
	if n.Config.Uid != "" {
		msg.Metadata.PutValue("uid", n.Config.Uid)
	}
	if !msg.Metadata.Has("uid") {
		user, err := store.Database.FirstUser()
		if err != nil {
			ctx.TellFailure(msg, err)
			return
		}
		if user != nil && user.ID > 0 {
			msg.Metadata.PutValue("uid", user.Flag)
		}
	}

	ctx.TellSuccess(msg)
	return
}

func (n *DefaultUserNode) Destroy() {}
