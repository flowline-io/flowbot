package ability

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
)

func TestOpKnownCapability(t *testing.T) {
	assert.Equal(t, "list", Op(hub.CapBookmark, "List"))
	assert.Equal(t, "get", Op(hub.CapBookmark, "Get"))
	assert.Equal(t, "create", Op(hub.CapBookmark, "Create"))
	assert.Equal(t, "delete", Op(hub.CapBookmark, "Delete"))
	assert.Equal(t, "archive", Op(hub.CapBookmark, "Archive"))
	assert.Equal(t, "search", Op(hub.CapBookmark, "Search"))
	assert.Equal(t, "attach_tags", Op(hub.CapBookmark, "AttachTags"))
	assert.Equal(t, "detach_tags", Op(hub.CapBookmark, "DetachTags"))
	assert.Equal(t, "check_url", Op(hub.CapBookmark, "CheckURL"))
}

func TestOpArchiveCapability(t *testing.T) {
	assert.Equal(t, "add", Op(hub.CapArchive, "Add"))
	assert.Equal(t, "search", Op(hub.CapArchive, "Search"))
	assert.Equal(t, "get", Op(hub.CapArchive, "Get"))
}

func TestOpReaderCapability(t *testing.T) {
	assert.Equal(t, "list_feeds", Op(hub.CapReader, "ListFeeds"))
	assert.Equal(t, "create_feed", Op(hub.CapReader, "CreateFeed"))
	assert.Equal(t, "list_entries", Op(hub.CapReader, "ListEntries"))
	assert.Equal(t, "mark_entry_read", Op(hub.CapReader, "MarkEntryRead"))
	assert.Equal(t, "mark_entry_unread", Op(hub.CapReader, "MarkEntryUnread"))
	assert.Equal(t, "star_entry", Op(hub.CapReader, "StarEntry"))
	assert.Equal(t, "unstar_entry", Op(hub.CapReader, "UnstarEntry"))
}

func TestOpKanbanCapability(t *testing.T) {
	assert.Equal(t, "list_tasks", Op(hub.CapKanban, "ListTasks"))
	assert.Equal(t, "get_task", Op(hub.CapKanban, "GetTask"))
	assert.Equal(t, "create_task", Op(hub.CapKanban, "CreateTask"))
	assert.Equal(t, "update_task", Op(hub.CapKanban, "UpdateTask"))
	assert.Equal(t, "delete_task", Op(hub.CapKanban, "DeleteTask"))
	assert.Equal(t, "move_task", Op(hub.CapKanban, "MoveTask"))
	assert.Equal(t, "complete_task", Op(hub.CapKanban, "CompleteTask"))
	assert.Equal(t, "get_columns", Op(hub.CapKanban, "GetColumns"))
	assert.Equal(t, "search_tasks", Op(hub.CapKanban, "SearchTasks"))
}

func TestOpFinanceCapability(t *testing.T) {
	assert.Equal(t, "create_transaction", Op(hub.CapFinance, "CreateTransaction"))
}

func TestOpUnknownCapability(t *testing.T) {
	assert.Equal(t, "", Op("unknown_cap", "SomeKey"))
}

func TestOpUnknownKey(t *testing.T) {
	assert.Equal(t, "", Op(hub.CapBookmark, "UnknownKey"))
}

func TestOpEmptyCapability(t *testing.T) {
	assert.Equal(t, "", Op("", "anything"))
}

func TestOperationsMapHasExpectedCapabilities(t *testing.T) {
	assert.Contains(t, Operations, hub.CapBookmark)
	assert.Contains(t, Operations, hub.CapArchive)
	assert.Contains(t, Operations, hub.CapReader)
	assert.Contains(t, Operations, hub.CapKanban)
	assert.Contains(t, Operations, hub.CapFinance)
}

func TestOperationsMapBookmarkKeys(t *testing.T) {
	ops := Operations[hub.CapBookmark]
	assert.Len(t, ops, 9)
	assert.Equal(t, "list", ops["List"])
	assert.Equal(t, "get", ops["Get"])
	assert.Equal(t, "create", ops["Create"])
}

func TestOperationConstantsMatchOperationsMap(t *testing.T) {
	assert.Equal(t, Operations[hub.CapBookmark]["List"], OpBookmarkList)
	assert.Equal(t, Operations[hub.CapBookmark]["Get"], OpBookmarkGet)
	assert.Equal(t, Operations[hub.CapBookmark]["Create"], OpBookmarkCreate)
	assert.Equal(t, Operations[hub.CapBookmark]["Delete"], OpBookmarkDelete)
	assert.Equal(t, Operations[hub.CapBookmark]["Archive"], OpBookmarkArchive)
	assert.Equal(t, Operations[hub.CapBookmark]["Search"], OpBookmarkSearch)
	assert.Equal(t, Operations[hub.CapBookmark]["AttachTags"], OpBookmarkAttachTags)
	assert.Equal(t, Operations[hub.CapBookmark]["DetachTags"], OpBookmarkDetachTags)
	assert.Equal(t, Operations[hub.CapBookmark]["CheckURL"], OpBookmarkCheckURL)

	assert.Equal(t, Operations[hub.CapArchive]["Add"], OpArchiveAdd)
	assert.Equal(t, Operations[hub.CapArchive]["Search"], OpArchiveSearch)
	assert.Equal(t, Operations[hub.CapArchive]["Get"], OpArchiveGet)

	assert.Equal(t, Operations[hub.CapReader]["ListFeeds"], OpReaderListFeeds)
	assert.Equal(t, Operations[hub.CapReader]["CreateFeed"], OpReaderCreateFeed)
	assert.Equal(t, Operations[hub.CapReader]["ListEntries"], OpReaderListEntries)
	assert.Equal(t, Operations[hub.CapReader]["MarkEntryRead"], OpReaderMarkEntryRead)

	assert.Equal(t, Operations[hub.CapKanban]["ListTasks"], OpKanbanListTasks)
	assert.Equal(t, Operations[hub.CapKanban]["CreateTask"], OpKanbanCreateTask)

	assert.Equal(t, Operations[hub.CapFinance]["CreateTransaction"], OpFinanceCreateTransaction)
}
