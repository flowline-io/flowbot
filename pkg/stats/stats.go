package stats

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
	"github.com/flowline-io/flowbot/version"
)

const (
	BotTotalStatsName      = "bot_total"
	BotRunTotalStatsName   = "bot_run_total"
	BookmarkTotalStatsName = "bookmark_total"
)

type RulesetLabel string

const (
	InputRuleset   RulesetLabel = "input"
	AgentRuleset   RulesetLabel = "agent"
	CommandRuleset RulesetLabel = "command"
	CronRuleset    RulesetLabel = "cron"
	FormRuleset    RulesetLabel = "form"
)

func BotTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, BotTotalStatsName, version.Buildtags))
}

func BotRunTotalCounter(rulesetLabel RulesetLabel) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{ruleset="%s",version="%s"}`, BotRunTotalStatsName, rulesetLabel, version.Buildtags))
}

func BookmarkTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, BookmarkTotalStatsName, version.Buildtags))
}
