package agent

import (
	"context"
	"fmt"
	"sync"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/tmc/langchaingo/llms"
)

// Agent is a stateful wrapper around the agent loop with queues and subscriptions.
type Agent struct {
	mu          sync.Mutex
	state       *Context
	cfg         Config
	deps        LoopDeps
	subscribers []agentevent.Handler
	steering    *messageQueue
	followUp    *messageQueue
	cancel      context.CancelFunc
}

// Options configures a new Agent instance.
type Options struct {
	InitialState *Context
	Config       Config
	Model        llms.Model
	Registry     *tool.Registry
}

// NewAgent creates an agent with default transforms and optional dependencies.
func NewAgent(opts Options) *Agent {
	cfg := opts.Config.WithDefaults()
	if cfg.ConvertToLLM == nil {
		cfg.ConvertToLLM = transform.DefaultConvertToLLM
	}
	if cfg.TransformContext == nil {
		cfg.TransformContext = transform.FilterContext
	}

	state := &Context{}
	if opts.InitialState != nil {
		state = cloneContext(opts.InitialState)
	}

	registry := opts.Registry
	if registry == nil {
		registry = tool.NewRegistry()
	}

	agent := &Agent{
		state: state,
		cfg:   cfg,
		deps: LoopDeps{
			Model:    opts.Model,
			Registry: registry,
		},
		steering: newMessageQueue(cfg.SteeringMode),
		followUp: newMessageQueue(cfg.FollowUpMode),
	}
	agent.cfg.GetSteeringMessages = agent.steering.Drain
	agent.cfg.GetFollowUpMessages = agent.followUp.Drain
	return agent
}

// Subscribe registers a lifecycle event handler.
func (a *Agent) Subscribe(handler agentevent.Handler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.subscribers = append(a.subscribers, handler)
}

// State returns a snapshot of the current agent context.
func (a *Agent) State() *Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneContext(a.state)
}

// ApplyState atomically mutates the agent's internal state using the provided function.
// This avoids the clone-modify-discard pattern when the caller needs to update state in place.
func (a *Agent) ApplyState(fn func(*Context)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fn(a.state)
}

// SetTools replaces the active tool registry contents.
func (a *Agent) SetTools(registry *tool.Registry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.deps.Registry = registry
}

// SetActiveTools restricts which registered tools are exposed to the model.
func (a *Agent) SetActiveTools(names []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.deps.Registry != nil {
		a.deps.Registry.SetActive(names)
	}
}

// Prompt starts a new loop turn with one or more user messages.
func (a *Agent) Prompt(ctx context.Context, prompts ...AgentMessage) (*agentevent.Stream, error) {
	return a.run(ctx, prompts, false)
}

// Continue resumes the loop from the current context.
func (a *Agent) Continue(ctx context.Context) (*agentevent.Stream, error) {
	return a.run(ctx, nil, true)
}

// Steer enqueues a message injected between inner-loop turns.
func (a *Agent) Steer(message AgentMessage) {
	a.steering.Enqueue(message)
}

// FollowUp enqueues a message injected after the inner loop completes.
func (a *Agent) FollowUp(message AgentMessage) {
	a.followUp.Enqueue(message)
}

// Abort cancels the in-flight loop, if any.
func (a *Agent) Abort() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) run(parent context.Context, prompts []AgentMessage, continuing bool) (*agentevent.Stream, error) {
	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		return nil, ErrAborted
	}
	ctx, cancel := context.WithCancel(parent)
	a.cancel = cancel
	state := cloneContext(a.state)
	cfg := a.cfg
	deps := a.deps
	subscribers := append([]agentevent.Handler(nil), a.subscribers...)
	a.mu.Unlock()

	stream := agentevent.NewStream(64)
	for _, handler := range subscribers {
		stream.Subscribe(handler)
	}

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				stream.End(nil, fmt.Errorf("agent: panic: %v", r))
			}
		}()
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()

		var (
			newMessages []AgentMessage
			err         error
		)
		if continuing {
			newMessages, err = RunLoopContinue(ctx, state, cfg, deps, stream)
		} else {
			newMessages, err = RunLoop(ctx, prompts, state, cfg, deps, stream)
		}
		if err == nil {
			a.mu.Lock()
			a.state.Messages = append(a.state.Messages, newMessages...)
			a.mu.Unlock()
		}
		stream.End(toInterfaceMessages(newMessages), err)
	}()

	return stream, nil
}

type messageQueue struct {
	mode     QueueMode
	messages []AgentMessage
	mu       sync.Mutex
}

func newMessageQueue(mode QueueMode) *messageQueue {
	if mode == "" {
		mode = QueueAll
	}
	return &messageQueue{mode: mode}
}

func (q *messageQueue) Enqueue(message AgentMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, message)
}

func (q *messageQueue) Drain() ([]AgentMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.messages) == 0 {
		return nil, nil
	}
	if q.mode == QueueOne {
		first := q.messages[0]
		q.messages = q.messages[1:]
		return []AgentMessage{first}, nil
	}
	drained := append([]AgentMessage(nil), q.messages...)
	q.messages = nil
	return drained, nil
}

func toInterfaceMessages(messages []AgentMessage) []any {
	result := make([]any, len(messages))
	for i, message := range messages {
		result[i] = message
	}
	return result
}

// SetModel swaps the underlying langchaingo model at runtime.
func (a *Agent) SetModel(model llms.Model) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.deps.Model = model
}
