package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	agentresult "github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RunKind distinguishes interactive runs from autonomous scheduled task runs.
type RunKind string

const (
	// RunKindInteractive is the default chat agent run initiated by a user message.
	RunKindInteractive RunKind = "interactive"
	// RunKindScheduled is an autonomous run triggered by a scheduled task.
	RunKindScheduled RunKind = "scheduled"
	// RunKindPipeline is an autonomous run triggered by a pipeline agent step.
	RunKindPipeline RunKind = "pipeline"
)

// RunRequest carries one user turn for the chat assistant.
type RunRequest struct {
	SessionID    string
	Text         string
	API          *APIRunOptions
	Kind         RunKind
	RunStartedAt time.Time
}

// ManualCompactionResult reports the outcome of a user-triggered compaction run.
type ManualCompactionResult struct {
	Compacted    bool
	TokensBefore int
	TokensAfter  int
}

// Service orchestrates chat assistant agent runs for direct chat sessions.
type Service struct{}

// NewService creates a chat agent service using current application config.
func NewService() *Service {
	return &Service{}
}

// Run executes one agent turn and returns the assistant reply text.
func (*Service) Run(ctx context.Context, req RunRequest, sink StreamSink) (string, error) {
	start := time.Now()
	req.RunStartedAt = start
	textLen := len(strings.TrimSpace(req.Text))

	if err := validateRunRequest(ctx, req); err != nil {
		return "", err
	}

	lock := sessionLock(req.SessionID)
	lock.Lock()
	defer lock.Unlock()

	if err := ensureSessionActive(ctx, req.SessionID); err != nil {
		flog.Warn("[chat-agent] run rejected after lock: session=%s: %v", req.SessionID, err)
		return "", err
	}

	h, err := getOrCreateHarness(ctx, req, textLen)
	if err != nil {
		return "", err
	}

	return executeRun(ctx, h, req, start, sink)
}

// RunAPI executes one agent turn for HTTP clients with SSE event publishing.
func (s *Service) RunAPI(ctx context.Context, req RunRequest, opts *APIRunOptions) error {
	if opts == nil || opts.Publisher == nil {
		_, err := s.Run(ctx, req, nil)
		return err
	}
	req.API = opts
	_, err := s.Run(ctx, req, nil)
	return err
}

// CompactSession force-compacts the current session branch without sending a user turn.
func (*Service) CompactSession(ctx context.Context, sessionID string) (*ManualCompactionResult, error) {
	if !agentllm.AgentEnabled(agentName) {
		return nil, fmt.Errorf("chat agent is disabled or model is not configured")
	}
	if err := ensureSessionActive(ctx, sessionID); err != nil {
		return nil, err
	}

	lock := sessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	if err := ensureSessionActive(ctx, sessionID); err != nil {
		return nil, err
	}

	h, err := getOrCreateHarness(ctx, RunRequest{SessionID: sessionID}, 0)
	if err != nil {
		return nil, err
	}
	if h == nil || h.ContextManager() == nil || h.Session() == nil {
		return nil, fmt.Errorf("chat agent context manager unavailable")
	}

	branch, err := h.Session().GetBranch(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("load session branch: %w", err)
	}
	before := h.ContextManager().GetContextUsage(branch).Tokens

	err = h.ContextManager().CompactAndReload(ctx, h.Session(), h.Agent(), ctxmgr.CompactOpts{Force: true})
	if err != nil {
		if agentresult.IsCode(err, "nothing_to_compact") {
			return &ManualCompactionResult{Compacted: false, TokensBefore: before, TokensAfter: before}, nil
		}
		return nil, err
	}

	branch, err = h.Session().GetBranch(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("reload compacted branch: %w", err)
	}
	after := h.ContextManager().GetContextUsage(branch).Tokens
	return &ManualCompactionResult{
		Compacted:    true,
		TokensBefore: before,
		TokensAfter:  after,
	}, nil
}

func validateRunRequest(ctx context.Context, req RunRequest) error {
	if !agentllm.AgentEnabled(agentName) {
		flog.Warn("[chat-agent] run rejected: agent disabled or model not configured session=%s", req.SessionID)
		return fmt.Errorf("chat agent is disabled or model is not configured")
	}
	if strings.TrimSpace(req.Text) == "" {
		flog.Debug("[chat-agent] run rejected: empty message session=%s", req.SessionID)
		return fmt.Errorf("empty message")
	}
	if err := ensureSessionActive(ctx, req.SessionID); err != nil {
		flog.Warn("[chat-agent] run rejected: session inactive session=%s: %v", req.SessionID, err)
		return err
	}
	return nil
}

func executeRun(ctx context.Context, h *harness.Harness, req RunRequest, start time.Time, sink StreamSink) (string, error) {
	if !req.RunStartedAt.IsZero() {
		h.SetRunStartedAt(req.RunStartedAt)
	} else {
		h.SetRunStartedAt(start)
	}
	stream, err := h.Prompt(ctx, agent.NewUserMessage(req.Text))
	if err != nil {
		return handlePromptError(req.SessionID, start, err)
	}

	waitCoalescer := startRunStreamCoalescer(ctx, req, stream, sink)
	if err := h.WaitIdle(ctx); err != nil {
		finishRunCoalescer(waitCoalescer)
		if isRunInterrupted(err) {
			releaseHarnessAfterRunAbort(h, req.SessionID)
		}
		return "", awaitRunError(req.SessionID, start, err)
	}
	finishRunCoalescer(waitCoalescer)

	result := h.LastRunResult()
	if result.Err != nil {
		flog.Error(fmt.Errorf("[chat-agent] agent loop session=%s: %w", req.SessionID, result.Err))
		return "", result.Err
	}

	reply := resolveAssistantReply(req.SessionID, start, result.Messages)
	var resources []ResourceRef
	if planID, title, ok := maybePersistPlan(ctx, req.SessionID, reply); ok {
		reply = AppendPlanLinkFooter(reply, planID, title)
		resources = []ResourceRef{FormatPlanResourceRef(planID, title)}
	}
	deliverRunResult(ctx, h, req, reply, sink, result.Messages, resources, time.Since(start))
	maybeGenerateSessionTitle(req.SessionID, req.Text, reply)

	flog.Debug("[chat-agent] harness finished session=%s reply_len=%d duration=%s",
		req.SessionID, len(reply), time.Since(start).Round(time.Millisecond))
	return reply, nil
}

func handlePromptError(sessionID string, start time.Time, err error) (string, error) {
	if errors.Is(err, agent.ErrAborted) || agentresult.IsCode(err, "busy") {
		flog.Info("[chat-agent] harness busy session=%s duration=%s", sessionID, time.Since(start).Round(time.Millisecond))
		return "Agent is busy, please try again shortly.", nil
	}
	flog.Error(fmt.Errorf("[chat-agent] harness prompt session=%s: %w", sessionID, err))
	return "", err
}

func startRunStreamCoalescer(ctx context.Context, req RunRequest, stream *agentevent.Stream, sink StreamSink) func() {
	if req.API != nil && req.API.Publisher != nil && stream != nil {
		return startAPIEventStream(ctx, stream.Events(), req.API.Publisher, apiStreamUpdateInterval)
	}
	if sink != nil && stream != nil {
		return startStreamCoalescer(ctx, stream.Events(), sink, streamUpdateInterval)
	}
	return nil
}

func finishRunCoalescer(wait func()) {
	if wait != nil {
		wait()
	}
}

func resolveAssistantReply(sessionID string, start time.Time, messages []any) string {
	reply := extractAssistantReply(messages)
	if reply != "" {
		return reply
	}
	flog.Warn("[chat-agent] empty assistant reply session=%s duration=%s messages=%d",
		sessionID, time.Since(start).Round(time.Millisecond), len(messages))
	return "I could not produce a reply."
}

func deliverRunResult(ctx context.Context, h *harness.Harness, req RunRequest, reply string, sink StreamSink, messages []any, resources []ResourceRef, runDuration time.Duration) {
	if req.API != nil && req.API.Publisher != nil {
		contextWindow := 0
		if h != nil {
			if cm := h.ContextManager(); cm != nil {
				contextWindow = cm.ContextWindow()
			}
		}
		publishFinalUsage(req.API.Publisher, messages, contextWindow)
		title := LoadSessionTitle(ctx, req.SessionID)
		_ = req.API.Publisher.Publish(StreamEvent{
			Type:       EventTypeDone,
			Text:       reply,
			Title:      title,
			Resources:  resources,
			DurationMs: runDuration.Milliseconds(),
		})
		return
	}
	if sink == nil {
		return
	}
	if err := sink.Flush(ctx, reply); err != nil {
		flog.Warn("[chat-agent] stream flush session=%s: %v", req.SessionID, err)
	}
}

func awaitRunError(sessionID string, start time.Time, err error) error {
	if errors.Is(err, context.Canceled) {
		flog.Info("[chat-agent] run cancelled session=%s duration=%s", sessionID, time.Since(start).Round(time.Millisecond))
		return err
	}
	flog.Error(fmt.Errorf("[chat-agent] stream await session=%s: %w", sessionID, err))
	return err
}

func isRunInterrupted(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// releaseHarnessAfterRunAbort stops an in-flight loop and waits for the pooled harness to
// return idle before the session lock is released. Without this, a follow-up Prompt can
// observe PhaseBusy when cancellation only unblocked WaitIdle via ctx.Done().
func releaseHarnessAfterRunAbort(h *harness.Harness, sessionID string) {
	if h == nil {
		return
	}
	h.Agent().Abort()
	drainCtx, cancel := context.WithTimeout(context.Background(), harnessDrainTimeout)
	defer cancel()
	if err := h.WaitIdle(drainCtx); err != nil {
		flog.Warn("[chat-agent] harness drain after abort session=%s: %v; evicting pool entry", sessionID, err)
		EvictHarnessPool(sessionID)
	}
}

func ensureSessionActive(ctx context.Context, sessionID string) error {
	if store.Database == nil {
		return fmt.Errorf("chat session store unavailable")
	}
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return fmt.Errorf("chat session not found")
		}
		return fmt.Errorf("load chat session: %w", err)
	}
	if sess.State == int(schema.ChatSessionClosed) {
		return fmt.Errorf("chat session closed")
	}
	return nil
}

func extractAssistantReply(messages []any) string {
	for i := len(messages) - 1; i >= 0; i-- {
		message, ok := messages[i].(agent.AgentMessage)
		if !ok {
			continue
		}
		assistant, ok := message.(msg.AssistantMessage)
		if !ok {
			continue
		}
		text := strings.TrimSpace(msg.AssistantDisplayText(assistant))
		if text != "" {
			return text
		}
	}
	return ""
}

func publishFinalUsage(publisher EventPublisher, messages []any, contextWindow int) {
	var prompt, completion, total int
	for _, raw := range messages {
		message, ok := raw.(agent.AgentMessage)
		if !ok {
			continue
		}
		assistant, ok := message.(msg.AssistantMessage)
		if !ok || assistant.Usage == nil {
			continue
		}
		prompt += assistant.Usage.PromptTokens
		completion += assistant.Usage.CompletionTokens
		total += assistant.Usage.TotalTokens
	}
	if total > 0 {
		percent := 0.0
		if contextWindow > 0 {
			percent = float64(total) / float64(contextWindow) * 100
		}
		PublishUsageEvent(publisher, prompt, completion, total, contextWindow, percent)
	}
}

func textFromParts(parts []msg.ContentPart) string {
	var b strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// NewModelForTest overrides model creation in unit tests.
var NewModelForTest = agentllm.GetOrCreateModel
