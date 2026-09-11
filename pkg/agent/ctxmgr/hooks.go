package ctxmgr

import (
	"context"
	"errors"

	"github.com/flowline-io/flowbot/pkg/agent/session"
)

// CompactReason classifies why a compaction pass ran.
type CompactReason string

// Compaction reason values passed to BeforeCompact.
const (
	CompactReasonManual    CompactReason = "manual"
	CompactReasonThreshold CompactReason = "threshold"
	CompactReasonOverflow  CompactReason = "overflow"
)

// ErrCompactionCancelled indicates a BeforeCompact hook cancelled summarization.
var ErrCompactionCancelled = errors.New("ctxmgr: compaction cancelled by hook")

// ErrTreeNavigationCancelled indicates a BeforeTree hook cancelled MoveTo.
var ErrTreeNavigationCancelled = errors.New("ctxmgr: tree navigation cancelled by hook")

// BeforeCompactEvent is passed to BeforeCompactFn after preparation succeeds.
type BeforeCompactEvent struct {
	Preparation   *CompactionPreparation
	BranchEntries []session.TreeEntry
	Reason        CompactReason
	WillRetry     bool
}

// BeforeCompactOutcome can cancel compaction or supply a custom summary.
type BeforeCompactOutcome struct {
	Cancel     bool
	Compaction *CompactionResult
}

// BeforeCompactFn runs after PrepareCompaction and before RunCompaction.
type BeforeCompactFn func(ctx context.Context, event BeforeCompactEvent) (*BeforeCompactOutcome, error)

// BeforeTreeEvent is passed to BeforeTreeFn before MoveTo persists a new leaf.
type BeforeTreeEvent struct {
	TargetEntryID      string
	OldLeafID          string
	CommonAncestorID   string
	EntriesToSummarize []session.TreeEntry
	// UserWantsSummary is true when the caller did not supply a summary and the
	// manager would generate one; false when the caller already provided summary text.
	UserWantsSummary bool
}

// BeforeTreeOutcome can cancel navigation or supply a custom summary.
type BeforeTreeOutcome struct {
	Cancel  bool
	Summary string
}

// BeforeTreeFn runs on MoveTo before summarization or applying a caller summary.
type BeforeTreeFn func(ctx context.Context, event BeforeTreeEvent) (*BeforeTreeOutcome, error)
