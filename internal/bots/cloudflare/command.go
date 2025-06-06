package cloudflare

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/cloudflare"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "cloudflare setting",
		Help:   `cloudflare setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return chatbot.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "cloudflare test",
		Help:   "Test",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			c1, _ := chatbot.SettingGet(ctx, Name, tokenSettingKey)
			tokenValue, _ := c1.StringValue()
			c2, _ := chatbot.SettingGet(ctx, Name, zoneIdSettingKey)
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
				flog.Error(err)
				return nil
			}
			return nil
		},
	},
}
