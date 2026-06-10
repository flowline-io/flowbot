package ability

import (
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Operations returns a capability-specific operation constant.
// All ability operations are defined here to avoid import namespace conflicts
// with internal/modules packages.
var Operations = map[hub.CapabilityType]map[string]string{
	hub.CapExample: {
		"List":   "list",
		"Get":    "get",
		"Create": "create",
		"Update": "update",
		"Delete": "delete",
		"Health": "health",
	},
	hub.CapBookmark: {
		"List":       "list",
		"Get":        "get",
		"Create":     "create",
		"Delete":     "delete",
		"Archive":    "archive",
		"Search":     "search",
		"AttachTags": "attach_tags",
		"DetachTags": "detach_tags",
		"CheckURL":   "check_url",
	},
	hub.CapArchive: {
		"Add":    "add",
		"Search": "search",
		"Get":    "get",
	},
	hub.CapReader: {
		"ListFeeds":       "list_feeds",
		"CreateFeed":      "create_feed",
		"ListEntries":     "list_entries",
		"MarkEntryRead":   "mark_entry_read",
		"MarkEntryUnread": "mark_entry_unread",
		"StarEntry":       "star_entry",
		"UnstarEntry":     "unstar_entry",
	},
	hub.CapKanban: {
		"ListTasks":    "list_tasks",
		"GetTask":      "get_task",
		"CreateTask":   "create_task",
		"UpdateTask":   "update_task",
		"DeleteTask":   "delete_task",
		"MoveTask":     "move_task",
		"CompleteTask": "complete_task",
		"GetColumns":   "get_columns",
		"SearchTasks":  "search_tasks",
	},
	hub.CapFinance: {
		"CreateTransaction": "create_transaction",
	},
	hub.CapForge: {
		"GetUser":        "get_user",
		"GetRepo":        "get_repo",
		"ListIssues":     "list_issues",
		"GetIssue":       "get_issue",
		"GetCommitDiff":  "get_commit_diff",
		"GetFileContent": "get_file_content",
	},
	hub.CapGithub: {
		"GetUser":           "get_user",
		"GetUserByLogin":    "get_user_by_login",
		"GetRepo":           "get_repo",
		"ListIssues":        "list_issues",
		"GetIssue":          "get_issue",
		"GetCommitDiff":     "get_commit_diff",
		"GetFileContent":    "get_file_content",
		"ListNotifications": "list_notifications",
		"ListReleases":      "list_releases",
	},
	hub.CapNotify: {
		"Send":   "send",
		"Digest": "digest",
	},
	hub.CapNote: {
		"List":       "list",
		"Get":        "get",
		"Create":     "create",
		"Update":     "update",
		"Delete":     "delete",
		"GetContent": "get_content",
		"SetContent": "set_content",
		"Search":     "search",
		"GetAppInfo": "get_app_info",
	},
	hub.CapMemo: {
		"List":   "list",
		"Get":    "get",
		"Create": "create",
		"Update": "update",
		"Delete": "delete",
		"Health": "health",
	},
}

// Op returns the string operation name for the given capability and operation key.
// Example: ability.Op(hub.CapBookmark, "List") returns "list".
func Op(capType hub.CapabilityType, key string) string {
	if ops, ok := Operations[capType]; ok {
		return ops[key]
	}
	return ""
}

// Example operations as package-level constants.
const (
	OpExampleList   = "list"
	OpExampleGet    = "get"
	OpExampleCreate = "create"
	OpExampleUpdate = "update"
	OpExampleDelete = "delete"
	OpExampleHealth = "health"
)

// Bookmark operations as package-level constants for direct use.
const (
	OpBookmarkList       = "list"
	OpBookmarkGet        = "get"
	OpBookmarkCreate     = "create"
	OpBookmarkDelete     = "delete"
	OpBookmarkArchive    = "archive"
	OpBookmarkSearch     = "search"
	OpBookmarkAttachTags = "attach_tags"
	OpBookmarkDetachTags = "detach_tags"
	OpBookmarkCheckURL   = "check_url"
)

// Archive operations as package-level constants.
const (
	OpArchiveAdd    = "add"
	OpArchiveSearch = "search"
	OpArchiveGet    = "get"
)

// Reader operations as package-level constants.
const (
	OpReaderListFeeds       = "list_feeds"
	OpReaderCreateFeed      = "create_feed"
	OpReaderListEntries     = "list_entries"
	OpReaderMarkEntryRead   = "mark_entry_read"
	OpReaderMarkEntryUnread = "mark_entry_unread"
	OpReaderStarEntry       = "star_entry"
	OpReaderUnstarEntry     = "unstar_entry"
)

// Kanban operations as package-level constants.
const (
	OpKanbanListTasks    = "list_tasks"
	OpKanbanGetTask      = "get_task"
	OpKanbanCreateTask   = "create_task"
	OpKanbanUpdateTask   = "update_task"
	OpKanbanDeleteTask   = "delete_task"
	OpKanbanMoveTask     = "move_task"
	OpKanbanCompleteTask = "complete_task"
	OpKanbanGetColumns   = "get_columns"
	OpKanbanSearchTasks  = "search_tasks"
)

// Finance operations as package-level constants.
const (
	OpFinanceCreateTransaction = "create_transaction"
)

// Github operations as package-level constants.
const (
	OpGithubGetUser           = "get_user"
	OpGithubGetUserByLogin    = "get_user_by_login"
	OpGithubGetRepo           = "get_repo"
	OpGithubListIssues        = "list_issues"
	OpGithubGetIssue          = "get_issue"
	OpGithubGetCommitDiff     = "get_commit_diff"
	OpGithubGetFileContent    = "get_file_content"
	OpGithubListNotifications = "list_notifications"
	OpGithubListReleases      = "list_releases"
)

// Forge operations as package-level constants.
const (
	OpForgeGetUser        = "get_user"
	OpForgeGetRepo        = "get_repo"
	OpForgeListIssues     = "list_issues"
	OpForgeGetIssue       = "get_issue"
	OpForgeGetCommitDiff  = "get_commit_diff"
	OpForgeGetFileContent = "get_file_content"
)

// Notify operations as package-level constants.
const (
	OpNotifySend   = "send"
	OpNotifyDigest = "digest"
)

// Note operations as package-level constants.
const (
	OpNoteList       = "list"
	OpNoteGet        = "get"
	OpNoteCreate     = "create"
	OpNoteUpdate     = "update"
	OpNoteDelete     = "delete"
	OpNoteGetContent = "get_content"
	OpNoteSetContent = "set_content"
	OpNoteSearch     = "search"
	OpNoteGetAppInfo = "get_app_info"
)

// Memo operations as package-level constants.
const (
	OpMemoList   = "list"
	OpMemoGet    = "get"
	OpMemoCreate = "create"
	OpMemoUpdate = "update"
	OpMemoDelete = "delete"
	OpMemoHealth = "health"
)

// mutationOps is the explicit set of operations that modify state.
// Add new mutation operations here when introducing write operations.
var mutationOps = map[string]bool{
	// Generic CRUD mutations
	"create": true, "delete": true, "update": true,
	// Bookmark mutations
	"archive": true, "attach_tags": true, "detach_tags": true,
	// Kanban mutations
	"move_task": true, "complete_task": true,
	// Reader mutations
	"create_feed":     true,
	"mark_entry_read": true, "mark_entry_unread": true,
	"star_entry": true, "unstar_entry": true,
	// Notify mutations
	"send": true,
	// Archive mutations
	"add": true,
	// Finance mutations
	"create_transaction": true,
}

// IsMutation reports whether the operation name indicates a write that modifies state.
func IsMutation(op string) bool {
	return mutationOps[op]
}
