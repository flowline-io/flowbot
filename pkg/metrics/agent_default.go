package metrics

import "sync"

var (
	agentCollectorMu sync.RWMutex
	defaultAgent     *AgentCollector
)

// SetDefaultAgentCollector installs the process-wide agent metrics collector.
func SetDefaultAgentCollector(c *AgentCollector) {
	agentCollectorMu.Lock()
	defer agentCollectorMu.Unlock()
	defaultAgent = c
}

// DefaultAgentCollector returns the process-wide agent metrics collector (may be nil).
func DefaultAgentCollector() *AgentCollector {
	agentCollectorMu.RLock()
	defer agentCollectorMu.RUnlock()
	return defaultAgent
}

// Agent returns the default collector or a no-op collector when unset.
func Agent() *AgentCollector {
	if c := DefaultAgentCollector(); c != nil {
		return c
	}
	return &AgentCollector{}
}
