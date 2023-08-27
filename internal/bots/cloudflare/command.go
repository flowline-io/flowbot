package cloudflare

import (
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/parser"
	"github.com/sysatom/flowbot/pkg/providers/cloudflare"
	"time"
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
		Define: "setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "test",
		Help:   "Test",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			c1, _ := bots.SettingGet(ctx, Name, tokenSettingKey)
			tokenValue, _ := c1.StringValue()
			c2, _ := bots.SettingGet(ctx, Name, zoneIdSettingKey)
			zoneIdValue, _ := c2.StringValue()

			if tokenValue == "" || zoneIdValue == "" {
				return types.TextMsg{Text: "config error"}
			}

			now := time.Now()
			startDate := now.Add(-24 * time.Hour).Format(time.RFC3339)
			endDate := now.Format(time.RFC3339)

			provider := cloudflare.NewCloudflare(tokenValue, zoneIdValue)
			_, err := provider.GetAnalytics(startDate, endDate)
			if err != nil {
				logs.Err.Println(err)
				return nil
			}
			return nil
		},
	},
}
