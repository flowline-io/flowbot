package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestRegisterOperationsAndOp(t *testing.T) {
	tests := []struct {
		name string
		cap  hub.CapabilityType
		ops  map[string]string
		key  string
		want string
	}{
		{
			name: "karakeep List returns list",
			cap:  hub.CapKarakeep,
			ops:  map[string]string{"List": "list", "Get": "get", "Create": "create"},
			key:  "List",
			want: "list",
		},
		{
			name: "example Add returns add",
			cap:  hub.CapExample,
			ops:  map[string]string{"Add": "add", "Search": "search", "Get": "get"},
			key:  "Add",
			want: "add",
		},
		{
			name: "miniflux ListFeeds returns list_feeds",
			cap:  hub.CapMiniflux,
			ops:  map[string]string{"ListFeeds": "list_feeds", "CreateFeed": "create_feed"},
			key:  "ListFeeds",
			want: "list_feeds",
		},
		{
			name: "unknown key returns empty",
			cap:  hub.CapKanboard,
			ops:  map[string]string{"ListTasks": "list_tasks"},
			key:  "UnknownKey",
			want: "",
		},
		{
			name: "empty key returns empty",
			cap:  hub.CapGitea,
			ops:  map[string]string{"GetUser": "get_user"},
			key:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterOperations(tt.cap, tt.ops)
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
		{"numeric capability string returns empty", "cap_123", "List", ""},
		{"completely bogus capability returns empty", "???#$%", "AnyOp", ""},
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
		{"empty capability with empty key returns empty", "", "", ""},
		{"empty capability with valid known key returns empty", "", "List", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Op(tt.cap, tt.key))
		})
	}
}

func TestOperationsMapAfterRegister(t *testing.T) {
	tests := []struct {
		name string
		cap  hub.CapabilityType
		ops  map[string]string
	}{
		{"registers karakeep", hub.CapKarakeep, map[string]string{"List": "list", "Get": "get"}},
		{"registers example", hub.CapExample, map[string]string{"Add": "add"}},
		{"registers miniflux", hub.CapMiniflux, map[string]string{"ListFeeds": "list_feeds"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterOperations(tt.cap, tt.ops)
			assert.Contains(t, Operations, tt.cap)
			assert.Equal(t, tt.ops, Operations[tt.cap])
		})
	}
}

func TestIsMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
		want      bool
	}{
		{"list is read", "list", false},
		{"get is read", "get", false},
		{"search is read", "search", false},
		{"check_url is read", "check_url", false},
		{"list_tasks is read", "list_tasks", false},
		{"get_columns is read", "get_columns", false},
		{"create is mutation", "create", true},
		{"delete is mutation", "delete", true},
		{"update is mutation", "update", true},
		{"move_task is mutation", "move_task", true},
		{"archive is mutation", "archive", true},
		{"attach_tags is mutation", "attach_tags", true},
		{"detach_tags is mutation", "detach_tags", true},
		{"complete_task is mutation", "complete_task", true},
		{"mark_entry_read is mutation", "mark_entry_read", true},
		{"mark_entry_unread is mutation", "mark_entry_unread", true},
		{"star_entry is mutation", "star_entry", true},
		{"unstar_entry is mutation", "unstar_entry", true},
		{"send is mutation", "send", true},
		{"add is mutation", "add", true},
		{"create_feed is mutation", "create_feed", true},
		{"create_transaction is mutation", "create_transaction", true},
		{"digest is read", "digest", false},
		{"health is read", "health", false},
		{"get_user is read", "get_user", false},
		{"set_content is mutation", "set_content", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsMutation(tt.operation)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOperationConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"bookmark List constant", OpBookmarkList, "list"},
		{"bookmark Create constant", OpBookmarkCreate, "create"},
		{"bookmark Archive constant", OpBookmarkArchive, "archive"},
		{"example Add constant", OpArchiveAdd, "add"},
		{"reader ListFeeds constant", OpReaderListFeeds, "list_feeds"},
		{"kanban ListTasks constant", OpKanbanListTasks, "list_tasks"},
		{"agent Run constant", OpAgentRun, "run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
