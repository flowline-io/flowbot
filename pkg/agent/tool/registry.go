package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// UpdateHandler receives incremental tool execution progress.
type UpdateHandler func(update string) error

// Tool is an executable capability exposed to the agent loop.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, id string, args map[string]any, onUpdate UpdateHandler) (msg.ToolResultMessage, error)
}

// Registry stores registered tools and an optional active allowlist.
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	active map[string]struct{}
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool and rejects duplicate names.
func (r *Registry) Register(t Tool) error {
	if t == nil {
		return fmt.Errorf("tool registry: nil tool")
	}
	name := t.Name()
	if name == "" {
		return fmt.Errorf("tool registry: empty tool name")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool registry: duplicate tool %q", name)
	}
	r.tools[name] = t
	return nil
}

// SetActive restricts the tools exposed to the model; nil or empty means all registered tools.
func (r *Registry) SetActive(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(names) == 0 {
		r.active = nil
		return
	}
	r.active = make(map[string]struct{}, len(names))
	for _, name := range names {
		r.active[name] = struct{}{}
	}
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// ActiveTools returns tools allowed for the current run.
func (r *Registry) ActiveTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.active) == 0 {
		tools := make([]Tool, 0, len(r.tools))
		for _, t := range r.tools {
			tools = append(tools, t)
		}
		return tools
	}

	tools := make([]Tool, 0, len(r.active))
	for name := range r.active {
		if t, ok := r.tools[name]; ok {
			tools = append(tools, t)
		}
	}
	return tools
}
