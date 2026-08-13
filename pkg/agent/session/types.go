package session

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// EntryType identifies a persisted session tree node.
type EntryType string

const (
	EntryMessage           EntryType = "message"
	EntryModelChange       EntryType = "model_change"
	EntryActiveToolsChange EntryType = "active_tools_change"
	EntryBranchSummary     EntryType = "branch_summary"
	EntryCompaction        EntryType = "compaction"
	EntryCustom            EntryType = "custom"
	EntryTurnTrace         EntryType = "turn_trace"
)

// TraceSection is one named payload in a turn_trace node.
type TraceSection struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Hash    string `json:"hash"`
}

// NewTraceSection builds a section with a SHA-256 hex hash of content.
func NewTraceSection(name, content string) TraceSection {
	sum := sha256.Sum256([]byte(content))
	return TraceSection{
		Name:    name,
		Content: content,
		Hash:    hex.EncodeToString(sum[:]),
	}
}

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
	Sections         []TraceSection   `json:"sections,omitempty"`
	AssembleMs       int64            `json:"assemble_ms,omitempty"`
}

// Context is the reconstructed runtime view of a branch path.
type Context struct {
	Messages    []msg.AgentMessage
	ModelName   string
	ActiveTools []string
}
