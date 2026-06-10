package harness

import (
	"context"
	"log/slog"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// Phase represents the harness lifecycle state.
type Phase string

const (
	// PhaseIdle means no structural or loop operation is running.
	PhaseIdle Phase = "idle"
	// PhaseBusy means an agent loop is in progress.
	PhaseBusy Phase = "busy"
)

// HookHandler handles harness-level lifecycle hooks.
type HookHandler func(context.Context, HookEvent) error

// HookEvent is a harness lifecycle notification.
type HookEvent struct {
	Type         string
	Messages     []agent.AgentMessage
	SystemPrompt string
	ModelName    string
	ActiveTools  []string
}

// Options configures a harness instance.
type Options struct {
	AgentOptions agent.Options
	Session      *session.Session
	Router       *model.Router
	SystemPrompt string
	ModelName    string
}

// Harness orchestrates agent loop, session tree, tools, and lifecycle hooks.
type Harness struct {
	mu           sync.Mutex
	phase        Phase
	idleCh       chan struct{}
	agent        *agent.Agent
	session      *session.Session
	registry     *tool.Registry
	router       *model.Router
	systemPrompt string
	modelName    string
	hooks        map[string][]HookHandler
}

// New creates a harness with optional session and router dependencies.
func New(opts Options) *Harness {
	registry := opts.AgentOptions.Registry
	if registry == nil {
		registry = tool.NewRegistry()
	}
	opts.AgentOptions.Registry = registry

	return &Harness{
		agent:        agent.NewAgent(opts.AgentOptions),
		session:      opts.Session,
		registry:     registry,
		router:       opts.Router,
		systemPrompt: opts.SystemPrompt,
		modelName:    opts.ModelName,
		hooks:        make(map[string][]HookHandler),
		phase:        PhaseIdle,
		idleCh:       make(chan struct{}),
	}
}

// On registers a harness hook handler for an event type.
func (h *Harness) On(eventType string, handler HookHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks[eventType] = append(h.hooks[eventType], handler)
}

// RegisterTool adds a tool to the harness registry.
func (h *Harness) RegisterTool(t tool.Tool) error {
	return h.registry.Register(t)
}

// SetActiveTools restricts exposed tools for the next run.
func (h *Harness) SetActiveTools(names []string) {
	h.registry.SetActive(names)
	h.agent.SetActiveTools(names)
	h.emitHook(context.Background(), HookEvent{Type: "tools_update", ActiveTools: names})
}

// SetModel updates the active model name.
func (h *Harness) SetModel(llmModel llms.Model, modelName string) {
	h.agent.SetModel(llmModel)
	h.modelName = modelName
	h.emitHook(context.Background(), HookEvent{Type: "model_update", ModelName: modelName})
}

// Prompt starts an agent run with optional session persistence.
func (h *Harness) Prompt(ctx context.Context, prompts ...agent.AgentMessage) (*agentevent.Stream, error) {
	if err := h.requireIdle(); err != nil {
		return nil, err
	}
	h.setPhase(PhaseBusy)

	if err := h.runHook(ctx, "before_agent_start", HookEvent{
		Type:         "before_agent_start",
		Messages:     prompts,
		SystemPrompt: h.systemPrompt,
		ModelName:    h.modelName,
	}); err != nil {
		h.setPhase(PhaseIdle)
		return nil, err
	}

	if h.modelName != "" {
		h.mu.Lock()
		h.agent.ApplyState(func(state *agent.Context) {
			state.SystemPrompt = transform.MergeSystemPrompt(state.SystemPrompt, h.systemPrompt)
			state.ModelName = h.modelName
			if h.router != nil {
				h.router.ApplyToContext(state, false)
			}
		})
		h.mu.Unlock()
	}

	stream, err := h.agent.Prompt(ctx, prompts...)
	if err != nil {
		h.setPhase(PhaseIdle)
		return nil, err
	}
	go h.watchStream(ctx, stream)
	return stream, nil
}

// Agent exposes the underlying stateful agent.
func (h *Harness) Agent() *agent.Agent {
	return h.agent
}

// Session exposes the optional session manager.
func (h *Harness) Session() *session.Session {
	return h.session
}

// WaitIdle blocks until the harness returns to idle after a Prompt run finishes persisting.
func (h *Harness) WaitIdle(ctx context.Context) error {
	for {
		h.mu.Lock()
		if h.phase == PhaseIdle {
			h.mu.Unlock()
			return nil
		}
		ch := h.idleCh
		h.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
		}
	}
}

func (h *Harness) isIdle() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.phase == PhaseIdle
}

func (h *Harness) watchStream(ctx context.Context, stream *agentevent.Stream) {
	defer h.setPhase(PhaseIdle)

	result, awaitErr := stream.Await(ctx)
	if awaitErr != nil {
		return
	}
	if err := h.runHook(ctx, "save_point", HookEvent{Type: "save_point"}); err != nil {
		slog.Warn("harness: save_point hook", "err", err)
	}
	if result.Err != nil || h.session == nil {
		return
	}

	parentID, _ := h.currentLeafID(ctx)
	for _, item := range result.Messages {
		message, ok := item.(agent.AgentMessage)
		if !ok {
			continue
		}
		entryID := uuid.NewString()
		if err := h.session.Append(ctx, session.TreeEntry{
			ID:       entryID,
			ParentID: parentID,
			Type:     session.EntryMessage,
			Message:  message,
		}); err != nil {
			slog.Warn("harness: persist session entry", "err", err)
			continue
		}
		parentID = entryID
	}
}

func (h *Harness) requireIdle() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.phase != PhaseIdle {
		return agent.ErrAborted
	}
	return nil
}

func (h *Harness) setPhase(phase Phase) {
	h.mu.Lock()
	h.phase = phase
	if phase == PhaseIdle {
		close(h.idleCh)
		h.idleCh = make(chan struct{})
	}
	h.mu.Unlock()
}

func (h *Harness) runHook(ctx context.Context, eventType string, event HookEvent) error {
	h.mu.Lock()
	handlers := append([]HookHandler(nil), h.hooks[eventType]...)
	h.mu.Unlock()
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (h *Harness) emitHook(ctx context.Context, event HookEvent) {
	_ = h.runHook(ctx, event.Type, event)
}

func (h *Harness) currentLeafID(ctx context.Context) (string, error) {
	if h.session == nil {
		return "", nil
	}
	store := h.session
	branch, err := store.GetBranch(ctx, "")
	if err != nil {
		return "", err
	}
	if len(branch) == 0 {
		return "", nil
	}
	return branch[len(branch)-1].ID, nil
}
