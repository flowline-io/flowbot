package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/utils"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: "agent token",
		Help:   `get access token`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// get token
			value, err := store.Database.ConfigGet(ctx.AsUser, "", fmt.Sprintf("agent:%s:token", ctx.AsUser))
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			idValue, ok := value.String("value")
			if !ok || idValue == "" {
				idValue, err = utils.GenerateRandomString(25)
				if err != nil {
					return types.TextMsg{Text: "generate error"}
				}
				idValue = strings.ToLower(idValue)
				// set token
				err = store.Database.ConfigSet(ctx.AsUser, "",
					fmt.Sprintf("agent:%s:token", ctx.AsUser), types.KV{
						"value": idValue,
					})
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "set token error"}
				}
				data := types.KV{}
				data["uid"] = ctx.AsUser.String()
				err = store.Database.ParameterSet(idValue, data, time.Now().AddDate(1, 0, 0))
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "set token error"}
				}
			}

			return types.TextMsg{Text: fmt.Sprintf("[One-year validity token] %s", idValue)}
		},
	},
	{
		Define: "agent reset token",
		Help:   `reset access token`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// get old token
			value, err := store.Database.ConfigGet(ctx.AsUser, "", fmt.Sprintf("agent:%s:token", ctx.AsUser))
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			idValue, ok := value.String("value")
			if !ok || idValue == "" {
				return nil
			}
			// expire old token
			err = store.Database.ParameterSet(idValue, types.KV{}, time.Now())
			if err != nil {
				return nil
			}

			// new token
			idValue, err = utils.GenerateRandomString(25)
			if err != nil {
				return types.TextMsg{Text: "generate error"}
			}
			idValue = strings.ToLower(idValue)
			// set token
			err = store.Database.ConfigSet(ctx.AsUser, "",
				fmt.Sprintf("agent:%s:token", ctx.AsUser), types.KV{
					"value": idValue,
				})
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "set token error"}
			}
			data := types.KV{}
			data["uid"] = ctx.AsUser.String()
			err = store.Database.ParameterSet(idValue, data, time.Now().AddDate(1, 0, 0))
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "set token error"}
			}

			return types.TextMsg{Text: fmt.Sprintf("[One-year validity token] %s", idValue)}
		},
	},
}
