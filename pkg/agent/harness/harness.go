package harness

import (
	"context"
	"errors"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	agentresult "github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/flowline-io/flowbot/pkg/flog"
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
	ContextUsage *ctxmgr.ContextUsage
}

// Options configures a harness instance.
type Options struct {
	AgentOptions   agent.Options
	Session        *session.Session
	Router         *model.Router
	SystemPrompt   string
	ModelName      string
	ContextManager *ctxmgr.Manager
}

// Harness orchestrates agent loop, session tree, tools, and lifecycle hooks.
type Harness struct {
	mu           sync.Mutex
	phase        Phase
	idleCh       chan struct{}
	lastResult   agentevent.Result
	agent        *agent.Agent
	session      *session.Session
	registry     *tool.Registry
	router       *model.Router
	systemPrompt string
	modelName    string
	ctxMgr       *ctxmgr.Manager
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
		ctxMgr:       opts.ContextManager,
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

// MoveTo switches the session leaf, auto-summarizing abandoned branches when configured.
func (h *Harness) MoveTo(ctx context.Context, entryID, summary string) error {
	if h.session == nil {
		return normalizeHarnessError("busy", "session unavailable", agent.ErrAborted)
	}
	if h.ctxMgr != nil {
		if err := h.ctxMgr.MoveTo(ctx, h.session, entryID, summary); err != nil {
			if agentresult.IsCode(err, "aborted") {
				return nil
			}
			return normalizeHarnessError("branch_summary", "branch navigation failed", err)
		}
		return nil
	}
	if err := h.session.MoveTo(ctx, entryID, summary); err != nil {
		return normalizeHarnessError("branch_summary", "branch navigation failed", err)
	}
	return nil
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

	if err := h.prepareContext(ctx); err != nil {
		h.setPhase(PhaseIdle)
		return nil, wrapPromptError(err)
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
	go func() {
		result := h.watchStream(ctx, stream, prompts, 0)
		h.storeRunResult(result)
		h.setPhase(PhaseIdle)
	}()
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

// ContextManager exposes the optional context budget manager.
func (h *Harness) ContextManager() *ctxmgr.Manager {
	return h.ctxMgr
}

// LastRunResult returns the final outcome after Prompt completes, including overflow retries.
func (h *Harness) LastRunResult() agentevent.Result {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastResult
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

func (h *Harness) storeRunResult(result agentevent.Result) {
	h.mu.Lock()
	h.lastResult = result
	h.mu.Unlock()
}

func (h *Harness) prepareContext(ctx context.Context) error {
	if h.ctxMgr == nil || h.session == nil {
		return nil
	}
	if err := h.ctxMgr.EnsureWithinBudget(ctx, h.session, h.agent); err != nil {
		return err
	}
	h.emitContextUsage(ctx)
	return nil
}

func (h *Harness) emitContextUsage(ctx context.Context) {
	if h.ctxMgr == nil || h.session == nil {
		return
	}
	path, err := h.session.GetBranch(ctx, "")
	if err != nil {
		flog.Warn("harness: context usage branch load code=%s: %v", agentresult.CodeOf(err), err)
		return
	}
	usage := h.ctxMgr.GetContextUsage(path)
	_ = h.runHook(ctx, "context_usage", HookEvent{
		Type:         "context_usage",
		ContextUsage: &usage,
	})
}

func (h *Harness) watchStream(ctx context.Context, stream *agentevent.Stream, prompts []agent.AgentMessage, retry int) agentevent.Result {
	result, awaitErr := stream.Await(ctx)
	if awaitErr != nil {
		return agentevent.Result{Err: awaitErr}
	}

	if result.Err != nil && retry == 0 && h.shouldRetryOverflow(result) {
		if compactErr := h.ctxMgr.CompactAndReload(ctx, h.session, h.agent, ctxmgr.CompactOpts{Force: true}); compactErr != nil {
			h.finishStream(ctx, result)
			return agentevent.Result{Messages: result.Messages, Err: errors.Join(result.Err, compactErr)}
		}
		h.emitHook(ctx, HookEvent{Type: "context_compacted"})
		h.emitContextUsage(ctx)
		retryStream, promptErr := h.agent.Prompt(ctx, prompts...)
		if promptErr != nil {
			h.finishStream(ctx, result)
			return result
		}
		return h.watchStream(ctx, retryStream, prompts, retry+1)
	}

	h.finishStream(ctx, result)
	return result
}

func (h *Harness) shouldRetryOverflow(result agentevent.Result) bool {
	if h.ctxMgr == nil || h.session == nil || !h.ctxMgr.Settings().Enabled {
		return false
	}
	messages := agentMessagesFromResult(result.Messages)
	return ctxmgr.IsOverflowResult(result.Err, messages, h.ctxMgr.ContextWindow())
}

func (h *Harness) finishStream(ctx context.Context, result agentevent.Result) {
	if err := h.runHook(ctx, "save_point", HookEvent{Type: "save_point"}); err != nil {
		flog.Warn("harness: save_point hook: %v", err)
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
			flog.Warn("harness: persist session entry code=%s: %v", agentresult.CodeOf(err), err)
			continue
		}
		parentID = entryID
	}
}

func agentMessagesFromResult(messages []any) []msg.AgentMessage {
	result := make([]msg.AgentMessage, 0, len(messages))
	for _, item := range messages {
		message, ok := item.(agent.AgentMessage)
		if ok {
			result = append(result, message)
		}
	}
	return result
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
