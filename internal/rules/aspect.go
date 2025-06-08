package rules

import (
	"github.com/rulego/rulego/api/types"
)

var (
	_ types.StartAspect = (*Aspect)(nil)
)

type Aspect struct{}

func (a *Aspect) Order() int {
	return 900
}

func (a *Aspect) New() types.Aspect {
	return &Aspect{}
}

func (a *Aspect) Type() string {
	return "user"
}

func (a *Aspect) PointCut(_ types.RuleContext, _ types.RuleMsg, _ string) bool {
	return true
}

func (a *Aspect) Start(_ types.RuleContext, msg types.RuleMsg) (types.RuleMsg, error) {
	return msg, nil
}
