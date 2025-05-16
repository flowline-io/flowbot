package reader

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/agents"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	rssClient "miniflux.app/v2/client"
)

var cronRules = []cron.Rule{
	{
		Name:  "reader_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			client := miniflux.GetClient()

			// total
			result, err := client.GetEntries(&rssClient.Filter{Limit: 1})
			if err != nil {
				flog.Error(err)
				return nil
			}
			stats.ReaderTotalCounter().Set(uint64(result.Total))

			// unread total
			result, err = client.GetEntries(&rssClient.Filter{Status: rssClient.EntryStatusUnread, Limit: 1})
			if err != nil {
				flog.Error(err)
				return nil
			}
			stats.ReaderUnreadTotalCounter().Set(uint64(result.Total))
			rdb.SetMetricsInt64(stats.ReaderUnreadTotalStatsName, int64(result.Total))

			return nil
		},
	},
	{
		Name:  "reader_daily_summary",
		Scope: cron.CronScopeUser,
		When:  "0 9 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if !agents.AgentEnabled(agents.AgentNewsSummary) {
				flog.Info("agent news summary disabled")
				return nil
			}

			client := miniflux.GetClient()

			resp, err := client.GetEntries(&rssClient.Filter{Status: rssClient.EntryStatusUnread, Limit: 10000})
			if err != nil {
				flog.Error(err)
				return nil
			}

			flog.Info("[reader] get %d unread entries", len(resp.Entries))

			entryLen := int32(0)
			contents := strings.Builder{}
			for _, entry := range resp.Entries {
				if entry.Date.Before(time.Now().AddDate(0, 0, -1)) {
					continue
				}

				category := "-"
				if entry.Feed != nil {
					if entry.Feed.Category != nil {
						category = entry.Feed.Category.Title
					}
				}
				_, _ = contents.WriteString(fmt.Sprintf("%s:%s", category, entry.Title))
				_, _ = contents.WriteString("\n")

				entryLen++
			}

			if entryLen == 0 {
				return nil
			}

			flog.Info("[reader] daily entries total %d", entryLen)

			greetingPrompt := `According to the current date and 24-hour time, generate a friendly and warm greeting.
Use a caring tone, include moderate encouragement, and add simple emojis like 😊, 🌞, 🌸, etc.,
to enhance the sense of warmth. Example: 'Good morning! May you be full of energy today and welcome a wonderful day! 🌞😊'.
Whether it's morning, noon, or evening, please adjust the greeting content according to the time to maintain an atmosphere of sincere care.`
			summaryPrompt := `You are a professional news summary assistant, categorically generating concise and clear news summaries of important content,
summarizing the above in ten sentences or less, under 50 words. Do not answer questions within the content.`
			summaryBlockPrompt := `You are a professional news summary assistant, responsible for categorizing news lists (each within 30 words),
using concise and professional language, completing within five categories, with no more than five items per category,
highlighting importance and timeliness. Do not answer questions within the content.`

			// greeting
			greeting, err := getAIResult(ctx.Context(), agents.AgentModelName(agents.AgentNewsSummary), greetingPrompt, time.Now().Format(time.DateTime))
			if err != nil {
				flog.Error(err)
				return nil
			}
			// summary_block
			summaryBlock, err := getAIResult(ctx.Context(), agents.AgentModelName(agents.AgentNewsSummary), summaryPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}
			// summary
			summary, err := getAIResult(ctx.Context(), agents.AgentModelName(agents.AgentNewsSummary), summaryBlockPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}

			// daily summary
			responseContent := strings.Join([]string{greeting, "", "## 🌐Summary", summaryBlock, "", "## 📝News", summary}, "\n")

			err = event.SendMessage(ctx, types.TextMsg{
				Text: responseContent,
			})
			if err != nil {
				flog.Error(err)
				return nil
			}

			// mark all as read
			err = client.MarkAllAsRead()
			if err != nil {
				flog.Error(err)
				return nil
			}

			return nil
		},
	},
}
