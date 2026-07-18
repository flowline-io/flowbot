package capability

import "github.com/flowline-io/flowbot/pkg/hub"

// Legacy operation name constants kept for call-site compatibility during the
// 1:1 provider migration. Prefer package-local constants such as karakeep.OpList.
const (
	OpExampleList   = "list"
	OpExampleGet    = "get"
	OpExampleCreate = "create"
	OpExampleUpdate = "update"
	OpExampleDelete = "delete"
	OpExampleHealth = "health"

	OpBookmarkList       = "list"
	OpBookmarkGet        = "get"
	OpBookmarkCreate     = "create"
	OpBookmarkDelete     = "delete"
	OpBookmarkArchive    = "archive"
	OpBookmarkSearch     = "search"
	OpBookmarkAttachTags = "attach_tags"
	OpBookmarkDetachTags = "detach_tags"
	OpBookmarkCheckURL   = "check_url"

	OpArchiveAdd    = "add"
	OpArchiveSearch = "search"
	OpArchiveGet    = "get"

	OpReaderListFeeds       = "list_feeds"
	OpReaderCreateFeed      = "create_feed"
	OpReaderListEntries     = "list_entries"
	OpReaderMarkEntryRead   = "mark_entry_read"
	OpReaderMarkEntryUnread = "mark_entry_unread"
	OpReaderStarEntry       = "star_entry"
	OpReaderUnstarEntry     = "unstar_entry"

	OpKanbanListTasks    = "list_tasks"
	OpKanbanGetTask      = "get_task"
	OpKanbanCreateTask   = "create_task"
	OpKanbanUpdateTask   = "update_task"
	OpKanbanDeleteTask   = "delete_task"
	OpKanbanMoveTask     = "move_task"
	OpKanbanCompleteTask = "complete_task"
	OpKanbanGetColumns   = "get_columns"
	OpKanbanSearchTasks  = "search_tasks"

	OpFinanceCreateTransaction = "create_transaction"
	OpFinanceAbout             = "about"
	OpFinanceCurrentUser       = "current_user"
	OpFinanceHealth            = "health"

	OpGithubGetUser           = "get_user"
	OpGithubGetUserByLogin    = "get_user_by_login"
	OpGithubGetRepo           = "get_repo"
	OpGithubListIssues        = "list_issues"
	OpGithubGetIssue          = "get_issue"
	OpGithubGetCommitDiff     = "get_commit_diff"
	OpGithubGetFileContent    = "get_file_content"
	OpGithubListNotifications = "list_notifications"
	OpGithubListReleases      = "list_releases"

	OpForgeGetUser        = "get_user"
	OpForgeGetRepo        = "get_repo"
	OpForgeListIssues     = "list_issues"
	OpForgeGetIssue       = "get_issue"
	OpForgeGetCommitDiff  = "get_commit_diff"
	OpForgeGetFileContent = "get_file_content"

	OpNotifySend   = "send"
	OpNotifyDigest = "digest"

	OpNoteList       = "list"
	OpNoteGet        = "get"
	OpNoteCreate     = "create"
	OpNoteUpdate     = "update"
	OpNoteDelete     = "delete"
	OpNoteGetContent = "get_content"
	OpNoteSetContent = "set_content"
	OpNoteSearch     = "search"
	OpNoteGetAppInfo = "get_app_info"

	OpMemoList   = "list"
	OpMemoGet    = "get"
	OpMemoCreate = "create"
	OpMemoUpdate = "update"
	OpMemoDelete = "delete"
	OpMemoHealth = "health"

	OpAgentRun = "run"
)

// Op looks up a registered operation name by capability and key.
func Op(capType hub.CapabilityType, key string) string {
	operationsMu.RLock()
	defer operationsMu.RUnlock()
	if ops, ok := Operations[capType]; ok {
		return ops[key]
	}
	return ""
}
