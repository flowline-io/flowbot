package capability

import (
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
		"send": true, "run": true, "add": true,
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
