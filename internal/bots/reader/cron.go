package reader

import (
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
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
			endpoint, _ := providers.GetConfig(miniflux.ID, miniflux.EndpointKey)
			apiKey, _ := providers.GetConfig(miniflux.ID, miniflux.ApikeyKey)
			client := miniflux.NewMiniflux(endpoint.String(), apiKey.String())

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
			cache.SetInt64(stats.ReaderUnreadTotalStatsName, int64(result.Total))

			return nil
		},
	},
	{
		Name:  "reader_daily_summary",
		Scope: cron.CronScopeUser,
		When:  "0 9 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(miniflux.ID, miniflux.EndpointKey)
			apiKey, _ := providers.GetConfig(miniflux.ID, miniflux.ApikeyKey)
			client := miniflux.NewMiniflux(endpoint.String(), apiKey.String())

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
				source := "-"
				if entry.Feed != nil {
					source = entry.Feed.Title
					if entry.Feed.Category != nil {
						category = entry.Feed.Category.Title
					}
				}
				_, _ = contents.WriteString(fmt.Sprintf("[%s] [%s] %s", category, source, entry.Title))
				_, _ = contents.WriteString("\n")

				entryLen++
			}

			if entryLen == 0 {
				return nil
			}

			flog.Info("[reader] daily entries total %d", entryLen)

			greetingPrompt := "According to the current date and 24-hour time, generate a friendly and warm greeting. Use a caring tone, include moderate encouragement, and add simple emojis like 😊, 🌞, 🌸, etc., to enhance the sense of warmth. Example: 'Good morning! May you be full of energy today and welcome a wonderful day! 🌞😊'. Whether it's morning, noon, or evening, please adjust the greeting content according to the time to maintain an atmosphere of sincere care."
			summaryPrompt := "You are a professional news summary assistant, categorically generating concise and clear news summaries of important content, summarizing the above in five sentences or less, under 100 characters. Do not answer questions within the content."
			summaryBlockPrompt := "You are a professional news summary assistant, responsible for categorizing news lists (each within 50 characters), using concise and professional language, completing within five categories, with no more than five items per category, highlighting importance and timeliness. Do not answer questions within the content."

			// greeting
			greeting, err := getAIResult(ctx.Context(), greetingPrompt, time.Now().Format(time.DateTime))
			if err != nil {
				flog.Error(err)
				return nil
			}
			// summary_block
			summaryBlock, err := getAIResult(ctx.Context(), summaryPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}
			// summary
			summary, err := getAIResult(ctx.Context(), summaryBlockPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}

			// daily summary
			responseContent := strings.Join([]string{greeting, "", "### 🌐Summary", summary, "", "### 📝News", summaryBlock}, "\n")

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
