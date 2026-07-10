package harness

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	agentresult "github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/transform"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
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

// Options configures a harness instance.
type Options struct {
	AgentOptions   agent.Options
	Session        *session.Session
	Router         *model.Router
	SystemPrompt   string
	ModelName      string
	ContextManager *ctxmgr.Manager
	Hooks          *hooks.Registry
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
	hookRegistry *hooks.Registry
	loopBaseCfg  agent.Config
}

// New creates a harness with optional session and router dependencies.
func New(opts Options) *Harness {
	registry := opts.AgentOptions.Registry
	if registry == nil {
		registry = tool.NewRegistry()
	}
	opts.AgentOptions.Registry = registry

	if opts.Router != nil {
		cfg := opts.AgentOptions.Config
		if cfg.ChatModel == "" {
			cfg.ChatModel = opts.Router.ChatModel
		}
		if cfg.ToolModel == "" {
			cfg.ToolModel = opts.Router.ToolModel
		}
		opts.AgentOptions.Config = cfg
		if opts.ModelName == "" {
			opts.ModelName = opts.Router.ChatModel
		}
	}

	hookRegistry := opts.Hooks
	if hookRegistry == nil {
		hookRegistry = hooks.NewRegistry()
	}

	agentInstance := agent.NewAgent(opts.AgentOptions)

	return &Harness{
		agent:        agentInstance,
		session:      opts.Session,
		registry:     registry,
		router:       opts.Router,
		systemPrompt: opts.SystemPrompt,
		modelName:    opts.ModelName,
		ctxMgr:       opts.ContextManager,
		hookRegistry: hookRegistry,
		loopBaseCfg:  agentInstance.Config(),
		phase:        PhaseIdle,
		idleCh:       make(chan struct{}),
	}
}

// Hooks exposes the typed hook registry for this harness instance.
func (h *Harness) Hooks() *hooks.Registry {
	return h.hookRegistry
}

// RegisterTool adds a tool to the harness registry.
func (h *Harness) RegisterTool(t tool.Tool) error {
	return h.registry.Register(t)
}

// SetActiveTools restricts exposed tools for the next run.
func (h *Harness) SetActiveTools(names []string) {
	h.registry.SetActive(names)
	h.agent.SetActiveTools(names)
	h.emitObservation(context.Background(), hooks.ObservationEvent{Type: hooks.EventToolsUpdate, ActiveTools: names})
}

// SetModel updates the active model name.
func (h *Harness) SetModel(llmModel llms.Model, modelName string) {
	h.agent.SetModel(llmModel)
	h.modelName = modelName
	h.emitObservation(context.Background(), hooks.ObservationEvent{Type: hooks.EventModelUpdate, ModelName: modelName})
}

// SetSystemPrompt replaces the harness system prompt used on subsequent runs.
func (h *Harness) SetSystemPrompt(systemPrompt string) {
	h.mu.Lock()
	h.systemPrompt = systemPrompt
	h.mu.Unlock()
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

	prompts = append([]agent.AgentMessage(nil), prompts...)
	startResult, err := h.hookRegistry.EmitBeforeAgentStart(ctx, hooks.BeforeAgentStartEvent{
		Messages:     prompts,
		SystemPrompt: h.systemPrompt,
		ModelName:    h.modelName,
	})
	if err != nil {
		h.setPhase(PhaseIdle)
		metrics.Agent().IncRunTotal("error")
		return nil, err
	}
	if startResult != nil {
		if startResult.Cancel {
			h.setPhase(PhaseIdle)
			metrics.Agent().IncRunTotal("cancelled")
			return nil, hooks.ErrRunCancelled
		}
		if startResult.Messages != nil {
			prompts = startResult.Messages
		}
		if startResult.SystemPrompt != nil {
			h.systemPrompt = *startResult.SystemPrompt
		}
	}

	if err := h.prepareContext(ctx); err != nil {
		h.setPhase(PhaseIdle)
		metrics.Agent().IncRunTotal("error")
		return nil, wrapPromptError(err)
	}

	routed := model.ApplyDefaultRouter(h.loopBaseCfg)
	bridged := hooks.BridgeConfig(ctx, h.hookRegistry, routed)
	h.agent.ApplyConfig(func(cfg *agent.Config) {
		steering := cfg.GetSteeringMessages
		followUp := cfg.GetFollowUpMessages
		hooks.MergeHookFields(cfg, &bridged)
		cfg.GetSteeringMessages = steering
		cfg.GetFollowUpMessages = followUp
	})

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
		metrics.Agent().IncRunTotal("error")
		return nil, err
	}
	go func() {
		runCtx, span := trace.StartSpan(ctx, "agent.run")
		defer span.End()
		result := h.watchStream(runCtx, stream, prompts, 0)
		switch {
		case result.Err == nil:
			metrics.Agent().IncRunTotal("ok")
		case errors.Is(result.Err, agent.ErrAborted), errors.Is(result.Err, context.Canceled):
			metrics.Agent().IncRunTotal("cancelled")
			trace.RecordError(runCtx, result.Err)
		default:
			metrics.Agent().IncRunTotal("error")
			trace.RecordError(runCtx, result.Err)
		}
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
	ctx, span := trace.StartSpan(ctx, "agent.compact")
	defer span.End()
	if err := h.ctxMgr.EnsureWithinBudget(ctx, h.session, h.agent); err != nil {
		metrics.Agent().IncCompact("error")
		trace.RecordError(ctx, err)
		return err
	}
	metrics.Agent().IncCompact("ok")
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
	h.emitObservation(ctx, hooks.ObservationEvent{
		Type: hooks.EventContextUsage,
		ContextUsage: &hooks.ContextUsageInfo{
			Tokens:        usage.Tokens,
			ContextWindow: usage.ContextWindow,
			Percent:       usage.Percent,
		},
	})
}

func (h *Harness) watchStream(ctx context.Context, stream *agentevent.Stream, prompts []agent.AgentMessage, level int) agentevent.Result {
	// Await with a detached context so a cancelled run ctx cannot race ahead of the
	// agent loop and overwrite the loop's terminal error (for example ErrAborted).
	result, awaitErr := stream.Await(context.Background())
	if awaitErr != nil {
		return agentevent.Result{Err: awaitErr}
	}

	if result.Err != nil && h.shouldRetryOverflow(result) {
		nextLevel := level + 1
		force := nextLevel >= 2
		if nextLevel > 2 {
			h.finishStream(ctx, result)
			return result
		}
		metrics.Agent().IncOverflowRetry(fmt.Sprintf("%d", nextLevel))
		ctx, span := trace.StartSpan(ctx, "agent.compact")
		compactErr := h.ctxMgr.CompactAndReload(ctx, h.session, h.agent, ctxmgr.CompactOpts{Force: force})
		if compactErr != nil {
			metrics.Agent().IncCompact("error")
			trace.RecordError(ctx, compactErr)
			span.End()
			h.finishStream(ctx, result)
			return agentevent.Result{Messages: result.Messages, Err: errors.Join(result.Err, compactErr)}
		}
		metrics.Agent().IncCompact("ok")
		span.End()
		h.emitObservation(ctx, hooks.ObservationEvent{Type: hooks.EventContextCompacted})
		h.emitContextUsage(ctx)
		retryStream, promptErr := h.agent.Prompt(ctx, prompts...)
		if promptErr != nil {
			h.finishStream(ctx, result)
			return result
		}
		return h.watchStream(ctx, retryStream, prompts, nextLevel)
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
	h.emitObservation(ctx, hooks.ObservationEvent{Type: hooks.EventSavePoint})
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

func (h *Harness) emitObservation(ctx context.Context, event hooks.ObservationEvent) {
	h.hookRegistry.EmitObservation(ctx, event, func(format string, args ...any) {
		flog.Warn(format, args...)
	})
}

func (h *Harness) currentLeafID(ctx context.Context) (string, error) {
	if h.session == nil {
		return "", nil
	}
	return h.session.LeafID(ctx)
}
