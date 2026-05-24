package ability

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/hub"
)

// Operations returns a capability-specific operation constant.
// All ability operations are defined here to avoid import namespace conflicts
// with internal/modules packages.
var Operations = map[hub.CapabilityType]map[string]string{
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
	hub.CapNotify: {
		"Send":   "send",
		"Digest": "digest",
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

var mutationVerbs = []string{
	"create", "delete", "update", "move",
	"archive", "attach", "detach", "complete",
	"mark", "star", "unstar",
	"send", "add",
}

// IsMutation reports whether the operation name indicates a write that modifies state.
func IsMutation(op string) bool {
	for _, v := range mutationVerbs {
		if strings.Contains(op, v) {
			return true
		}
	}
	return false
}
