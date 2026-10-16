package capability

import (
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/hub"
)

var (
	operationsMu sync.RWMutex
	// Operations maps capability type to operation name set (populated by Register).
	Operations  = map[hub.CapabilityType]map[string]string{}
	mutationMu  sync.RWMutex
	mutationOps = map[string]bool{
		"create": true, "delete": true, "update": true,
		"archive": true, "attach_tags": true, "detach_tags": true,
		"move_task": true, "complete_task": true,
		"create_feed":     true,
		"mark_entry_read": true, "mark_entry_unread": true,
		"star_entry": true, "unstar_entry": true,
		"notify_send": true, "agent_run": true, "add": true,
		"http_request": true, "run_code": true, "run_terminal": true,
		"kv_set": true, "kv_delete": true, "clip_create": true,
		"create_transaction": true, "set_content": true,
	}
)

// RegisterOperations records operation names for a capability (used by UI/audit).
func RegisterOperations(capType hub.CapabilityType, ops map[string]string) {
	operationsMu.Lock()
	defer operationsMu.Unlock()
	Operations[capType] = ops
}

func registerMutation(op string) {
	mutationMu.Lock()
	defer mutationMu.Unlock()
	mutationOps[op] = true
}

// IsMutation reports whether the operation name indicates a write that modifies state.
func IsMutation(op string) bool {
	mutationMu.RLock()
	defer mutationMu.RUnlock()
	return mutationOps[op]
}

// Tier is the MCP annotation class for an operation.
type Tier int

const (
	// TierRead does not modify state.
	TierRead Tier = iota
	// TierWrite modifies state and can be repeated or reversed without deleting data.
	TierWrite
	// TierDestructive deletes data or removes a resource.
	TierDestructive
)

// Classify reports the MCP tier and whether a write is naturally idempotent.
// group names the tool group. The tier does not vary by group.
func Classify(group, op string) (Tier, bool) {
	tier, idempotent := classifyOp(op)
	if tier == TierRead && group == "" {
		return TierRead, false
	}
	return tier, idempotent
}

func classifyOp(op string) (Tier, bool) {
	if destructiveOp(op) {
		return TierDestructive, false
	}
	if idempotentOp(op) || op == "stop" || op == "run" || IsMutation(op) {
		return TierWrite, idempotentOp(op)
	}
	return TierRead, false
}

func destructiveOp(op string) bool {
	switch op {
	case "delete", "kv_delete", "archive", "remove":
		return true
	default:
		return strings.HasPrefix(op, "delete_")
	}
}

func idempotentOp(op string) bool {
	switch op {
	case "update", "set_content", "kv_set", "attach_tags", "detach_tags":
		return true
	default:
		return strings.HasPrefix(op, "update_") ||
			strings.HasPrefix(op, "mark_") ||
			strings.HasPrefix(op, "star_") ||
			strings.HasPrefix(op, "unstar_")
	}
}
