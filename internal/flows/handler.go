package flows

import (
	"sync"

	"github.com/flowline-io/flowbot/pkg/types"
)

// NodeHandler handles node execution
type NodeHandler interface {
	Trigger(ctx types.Context, params types.KV, variables types.KV) (types.KV, error)
	Action(ctx types.Context, params types.KV, variables types.KV) (types.KV, error)
}

var (
	handlers     = make(map[string]map[string]NodeHandler)
	handlersLock sync.RWMutex
)

// RegisterHandler registers a bot handler
func RegisterHandler(bot, ruleID string, handler NodeHandler) {
	handlersLock.Lock()
	defer handlersLock.Unlock()

	if handlers[bot] == nil {
		handlers[bot] = make(map[string]NodeHandler)
	}
	handlers[bot][ruleID] = handler
}

// getBotHandler gets a bot handler
func getBotHandler(bot, ruleID string) NodeHandler {
	handlersLock.RLock()
	defer handlersLock.RUnlock()

	if handlers[bot] == nil {
		return nil
	}
	return handlers[bot][ruleID]
}
