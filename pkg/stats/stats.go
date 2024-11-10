package stats

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
	"github.com/flowline-io/flowbot/version"
)

const (
	BotTotalStatsName                 = "bot_total"
	BotRunTotalStatsName              = "bot_run_total"
	BookmarkTotalStatsName            = "bookmark_total"
	SearchTotalStatsName              = "search_total"
	SearchAddDocumentTotalStatsName   = "search_add_document_total"
	QueueProcessedTasksTotalStatsName = "queue_processed_tasks_total"
	QueueFailedTasksTotalStatsName    = "queue_failed_tasks_total"
	QueueInProgressTasksStatsName     = "queue_in_progress_tasks"
	EventTotalStatsName               = "event_total"
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
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",ruleset="%s"}`, BotRunTotalStatsName, version.Buildtags, rulesetLabel))
}

func BookmarkTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, BookmarkTotalStatsName, version.Buildtags))
}

func SearchTotalCounter(index string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",index="%s"}`, SearchTotalStatsName, version.Buildtags, index))
}

func SearchAddDocumentTotalCounter(index string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",index="%s"}`, SearchAddDocumentTotalStatsName, version.Buildtags, index))
}

func QueueProcessedTasksTotalCounter(taskType string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",task_type="%s"}`, QueueProcessedTasksTotalStatsName, version.Buildtags, taskType))
}

func QueueFailedTasksTotalCounter(taskType string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",task_type="%s"}`, QueueFailedTasksTotalStatsName, version.Buildtags, taskType))
}

func QueueInProgressTasksCounter(taskType string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",task_type="%s"}`, QueueInProgressTasksStatsName, version.Buildtags, taskType))
}

func EventTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, EventTotalStatsName, version.Buildtags))
}
