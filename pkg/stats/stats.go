package stats

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
	"github.com/flowline-io/flowbot/version"
)

const (
	BotTotalStatsName                     = "bot_total"
	BotRunTotalStatsName                  = "bot_run_total"
	BookmarkTotalStatsName                = "bookmark_total"
	SearchTotalStatsName                  = "search_total"
	SearchProcessedDocumentTotalStatsName = "search_processed_document_total"
	QueueProcessedTasksTotalStatsName     = "queue_processed_tasks_total"
	QueueFailedTasksTotalStatsName        = "queue_failed_tasks_total"
	QueueInProgressTasksStatsName         = "queue_in_progress_tasks"
	EventTotalStatsName                   = "event_total"
	TorrentDownloadTotalStatsName         = "torrent_download_total"
	TorrentStatusTotalStatsName           = "torrent_status_total"
	GiteaIssueTotalStatsName              = "gitea_issue_total"
	KanbanEventTotalStatsName             = "kanban_event_total"
	KanbanTaskTotalStatsName              = "kanban_task_total"
	ReaderTotalStatsName                  = "reader_total"
	ReaderUnreadTotalStatsName            = "reader_unread_total"
	MonitorUpTotalStatsName               = "monitor_up_total"
	MonitorDownTotalStatsName             = "monitor_down_total"
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

func SearchProcessedDocumentTotalCounter(index string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",index="%s"}`, SearchProcessedDocumentTotalStatsName, version.Buildtags, index))
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

func TorrentDownloadTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, TorrentDownloadTotalStatsName, version.Buildtags))
}

func TorrentStatusTotalCounter(status string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",status="%s"}`, TorrentStatusTotalStatsName, version.Buildtags, status))
}

func GiteaIssueTotalCounter(status string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",status="%s"}`, GiteaIssueTotalStatsName, version.Buildtags, status))
}

func KanbanEventTotalCounter(name string) *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s",event_name="%s"}`, KanbanEventTotalStatsName, version.Buildtags, name))
}

func KanbanTaskTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, KanbanTaskTotalStatsName, version.Buildtags))
}

func ReaderTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, ReaderTotalStatsName, version.Buildtags))
}

func ReaderUnreadTotalCounter() *metrics.Counter {
	return metrics.GetOrCreateCounter(fmt.Sprintf(`%s{version="%s"}`, ReaderUnreadTotalStatsName, version.Buildtags))
}
