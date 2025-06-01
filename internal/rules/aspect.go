package rules

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/rulego/rulego/api/types"
	"gorm.io/gorm"
)

var (
	_ types.StartAspect = (*UserAspect)(nil)
)

type UserAspect struct{}

func (a *UserAspect) Order() int {
	return 900
}

func (a *UserAspect) New() types.Aspect {
	return &UserAspect{}
}

func (a *UserAspect) Type() string {
	return "user"
}

func (a *UserAspect) PointCut(_ types.RuleContext, _ types.RuleMsg, _ string) bool {
	return true
}

func (a *UserAspect) Start(_ types.RuleContext, msg types.RuleMsg) (types.RuleMsg, error) {
	if !msg.Metadata.Has("uid") {
		user, err := store.Database.FirstUser()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
		}
		if user != nil && user.ID > 0 {
			msg.Metadata.PutValue("uid", user.Flag)
			flog.Info("user aspect: %s set uid: %s", msg.Id, user.Flag)
		}
	}

	return msg, nil
}
