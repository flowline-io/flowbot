package ctxmgr

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// PrepareOptions configures compaction preparation behavior.
type PrepareOptions struct {
	// Force allows re-compaction when the session leaf is already a compaction node.
	Force bool
	// ExtraMessages are unpersisted agent messages appended to the summarization set.
	ExtraMessages []msg.AgentMessage
}

// CompactOpts configures a compaction run invoked by the manager.
type CompactOpts struct {
	// Force enables overflow-style re-compaction after a prior compaction leaf.
	Force bool
}
