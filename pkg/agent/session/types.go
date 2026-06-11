package session

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// EntryType identifies a persisted session tree node.
type EntryType string

const (
	EntryMessage           EntryType = "message"
	EntryModelChange       EntryType = "model_change"
	EntryActiveToolsChange EntryType = "active_tools_change"
	EntryBranchSummary     EntryType = "branch_summary"
	EntryCompaction        EntryType = "compaction"
	EntryCustom            EntryType = "custom"
)

// TreeEntry is one node in the append-only session tree.
type TreeEntry struct {
	ID       string    `json:"id"`
	ParentID string    `json:"parent_id"`
	Type     EntryType `json:"type"`

	Message          msg.AgentMessage `json:"-"`
	MessageRaw       any              `json:"message,omitempty"`
	ModelName        string           `json:"model_name,omitempty"`
	ActiveToolNames  []string         `json:"active_tool_names,omitempty"`
	Summary          string           `json:"summary,omitempty"`
	FromID           string           `json:"from_id,omitempty"`
	FirstKeptEntryID string           `json:"first_kept_entry_id,omitempty"`
	TokensBefore     int              `json:"tokens_before,omitempty"`
	ReadFiles        []string         `json:"read_files,omitempty"`
	ModifiedFiles    []string         `json:"modified_files,omitempty"`
}

// Context is the reconstructed runtime view of a branch path.
type Context struct {
	Messages    []msg.AgentMessage
	ModelName   string
	ActiveTools []string
}
