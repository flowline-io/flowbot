package reader

import (
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/llm"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "reader_metrics",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderListEntries, map[string]any{})
			if err != nil {
				flog.Error(err)
				return nil
			}
			entries, _ := res.Data.([]*ability.Entry)

			total := len(entries)
			stats.ReaderTotalCounter().Set(uint64(total))

			unreadCount := 0
			for _, e := range entries {
				if e.Status == "unread" {
					unreadCount++
				}
			}
			stats.ReaderUnreadTotalCounter().Set(uint64(unreadCount))
			rdb.SetMetricsInt64(stats.ReaderUnreadTotalStatsName, int64(unreadCount))

			return nil
		},
	},
	{
		Name:  "reader_daily_summary",
		Scope: cron.CronScopeSystem,
		When:  "0 9 * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			if !llm.AgentEnabled(llm.AgentNewsSummary) {
				flog.Info("agent news summary disabled")
				return nil
			}

			res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderListEntries, map[string]any{
				"status": "unread",
			})
			if err != nil {
				flog.Error(err)
				return nil
			}

			entries, _ := res.Data.([]*ability.Entry)
			flog.Info("[reader] get %d unread entries", len(entries))

			entryLen := int32(0)
			contents := strings.Builder{}
			for _, entry := range entries {
				if entry.PublishedAt.Before(time.Now().AddDate(0, 0, -1)) {
					continue
				}

				category := entry.FeedTitle
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

			greeting, err := getAIResult(ctx.Context(), llm.AgentModelName(llm.AgentNewsSummary), greetingPrompt, time.Now().Format(time.DateTime))
			if err != nil {
				flog.Error(err)
				return nil
			}
			summaryBlock, err := getAIResult(ctx.Context(), llm.AgentModelName(llm.AgentNewsSummary), summaryPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}
			summary, err := getAIResult(ctx.Context(), llm.AgentModelName(llm.AgentNewsSummary), summaryBlockPrompt, contents.String())
			if err != nil {
				flog.Error(err)
				return nil
			}

			responseContent := strings.Join([]string{greeting, "", "## Summary", summaryBlock, "", "## News", summary}, "\n")

			err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "reader.news.summary", []string{"slack", "ntfy"}, map[string]any{
				"body":        responseContent,
				"entry_count": entryLen,
			})
			if err != nil {
				flog.Error(err)
				return nil
			}

			for _, entry := range entries {
				if entry.Status == "unread" {
					_, err = ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderMarkEntryRead, map[string]any{
						"id": entry.ID,
					})
					if err != nil {
						flog.Error(err)
					}
				}
			}

			return nil
		},
	},
}
