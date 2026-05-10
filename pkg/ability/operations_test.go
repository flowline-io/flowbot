package ability

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestOpKnownCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"List returns list", hub.CapBookmark, "List", "list"},
		{"Get returns get", hub.CapBookmark, "Get", "get"},
		{"Create returns create", hub.CapBookmark, "Create", "create"},
		{"Delete returns delete", hub.CapBookmark, "Delete", "delete"},
		{"Archive returns archive", hub.CapBookmark, "Archive", "archive"},
		{"Search returns search", hub.CapBookmark, "Search", "search"},
		{"AttachTags returns attach_tags", hub.CapBookmark, "AttachTags", "attach_tags"},
		{"DetachTags returns detach_tags", hub.CapBookmark, "DetachTags", "detach_tags"},
		{"CheckURL returns check_url", hub.CapBookmark, "CheckURL", "check_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpArchiveCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"Add returns add", hub.CapArchive, "Add", "add"},
		{"Search returns search", hub.CapArchive, "Search", "search"},
		{"Get returns get", hub.CapArchive, "Get", "get"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpReaderCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"ListFeeds returns list_feeds", hub.CapReader, "ListFeeds", "list_feeds"},
		{"CreateFeed returns create_feed", hub.CapReader, "CreateFeed", "create_feed"},
		{"ListEntries returns list_entries", hub.CapReader, "ListEntries", "list_entries"},
		{"MarkEntryRead returns mark_entry_read", hub.CapReader, "MarkEntryRead", "mark_entry_read"},
		{"MarkEntryUnread returns mark_entry_unread", hub.CapReader, "MarkEntryUnread", "mark_entry_unread"},
		{"StarEntry returns star_entry", hub.CapReader, "StarEntry", "star_entry"},
		{"UnstarEntry returns unstar_entry", hub.CapReader, "UnstarEntry", "unstar_entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpKanbanCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"ListTasks returns list_tasks", hub.CapKanban, "ListTasks", "list_tasks"},
		{"GetTask returns get_task", hub.CapKanban, "GetTask", "get_task"},
		{"CreateTask returns create_task", hub.CapKanban, "CreateTask", "create_task"},
		{"UpdateTask returns update_task", hub.CapKanban, "UpdateTask", "update_task"},
		{"DeleteTask returns delete_task", hub.CapKanban, "DeleteTask", "delete_task"},
		{"MoveTask returns move_task", hub.CapKanban, "MoveTask", "move_task"},
		{"CompleteTask returns complete_task", hub.CapKanban, "CompleteTask", "complete_task"},
		{"GetColumns returns get_columns", hub.CapKanban, "GetColumns", "get_columns"},
		{"SearchTasks returns search_tasks", hub.CapKanban, "SearchTasks", "search_tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpFinanceCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"CreateTransaction returns create_transaction", hub.CapFinance, "CreateTransaction", "create_transaction"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpUnknownCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"unknown capability returns empty", "unknown_cap", "SomeKey", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpUnknownKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"unknown key returns empty", hub.CapBookmark, "UnknownKey", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOpEmptyCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"empty capability returns empty", "", "anything", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOperationsMapHasExpectedCapabilities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
	}{
		{"has bookmark", hub.CapBookmark},
		{"has archive", hub.CapArchive},
		{"has reader", hub.CapReader},
		{"has kanban", hub.CapKanban},
		{"has finance", hub.CapFinance},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, Operations, tt.cap)
		})
	}
}

func TestOperationsMapBookmarkKeys(t *testing.T) {
	t.Parallel()
	ops := Operations[hub.CapBookmark]

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"List key equals list", "List", "list"},
		{"Get key equals get", "Get", "get"},
		{"Create key equals create", "Create", "create"},
	}
	t.Run("has correct number of operations", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, ops, 9)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ops[tt.key])
		})
	}
}

func TestOperationConstantsMatchOperationsMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cap  hub.CapabilityType
		key  string
		want string
	}{
		{"bookmark List constant matches", hub.CapBookmark, "List", OpBookmarkList},
		{"bookmark Get constant matches", hub.CapBookmark, "Get", OpBookmarkGet},
		{"bookmark Create constant matches", hub.CapBookmark, "Create", OpBookmarkCreate},
		{"bookmark Delete constant matches", hub.CapBookmark, "Delete", OpBookmarkDelete},
		{"bookmark Archive constant matches", hub.CapBookmark, "Archive", OpBookmarkArchive},
		{"bookmark Search constant matches", hub.CapBookmark, "Search", OpBookmarkSearch},
		{"bookmark AttachTags constant matches", hub.CapBookmark, "AttachTags", OpBookmarkAttachTags},
		{"bookmark DetachTags constant matches", hub.CapBookmark, "DetachTags", OpBookmarkDetachTags},
		{"bookmark CheckURL constant matches", hub.CapBookmark, "CheckURL", OpBookmarkCheckURL},
		{"archive Add constant matches", hub.CapArchive, "Add", OpArchiveAdd},
		{"archive Search constant matches", hub.CapArchive, "Search", OpArchiveSearch},
		{"archive Get constant matches", hub.CapArchive, "Get", OpArchiveGet},
		{"reader ListFeeds constant matches", hub.CapReader, "ListFeeds", OpReaderListFeeds},
		{"reader CreateFeed constant matches", hub.CapReader, "CreateFeed", OpReaderCreateFeed},
		{"reader ListEntries constant matches", hub.CapReader, "ListEntries", OpReaderListEntries},
		{"reader MarkEntryRead constant matches", hub.CapReader, "MarkEntryRead", OpReaderMarkEntryRead},
		{"kanban ListTasks constant matches", hub.CapKanban, "ListTasks", OpKanbanListTasks},
		{"kanban CreateTask constant matches", hub.CapKanban, "CreateTask", OpKanbanCreateTask},
		{"finance CreateTransaction constant matches", hub.CapFinance, "CreateTransaction", OpFinanceCreateTransaction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Operations[tt.cap][tt.key])
		})
	}
}
