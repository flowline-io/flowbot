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
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RunRequest carries one user turn for the chat assistant.
type RunRequest struct {
	SessionID string
	Text      string
}

// Service orchestrates chat assistant agent runs for direct chat sessions.
type Service struct{}

// NewService creates a chat agent service using current application config.
func NewService() *Service {
	return &Service{}
}

// Run executes one agent turn and returns the assistant reply text.
func (*Service) Run(ctx context.Context, req RunRequest) (string, error) {
	start := time.Now()
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

	h, err := newRunHarness(ctx, req, textLen)
	if err != nil {
		return "", err
	}

	return executeRun(ctx, h, req, start)
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

func newRunHarness(ctx context.Context, req RunRequest, textLen int) (*harness.Harness, error) {
	workspace, err := WorkspaceFromConfig()
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] workspace config session=%s: %w", req.SessionID, err))
		return nil, err
	}

	modelName := agentllm.AgentModelName(agentName)
	llmModel, resolvedName, err := NewModelForTest(ctx, modelName)
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] model init session=%s model=%s: %w", req.SessionID, modelName, err))
		return nil, fmt.Errorf("chat agent model: %w", err)
	}

	registry, err := NewRegistry(workspace)
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] tool registry session=%s: %w", req.SessionID, err))
		return nil, err
	}

	agentSession := session.New(NewDBStorage(req.SessionID))
	branch, err := agentSession.GetBranch(ctx, "")
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] load branch session=%s: %w", req.SessionID, err))
		return nil, fmt.Errorf("load session branch: %w", err)
	}

	systemPrompt := SystemPrompt(ctx, workspace)
	agentCtx := session.ToAgentContext(session.BuildContext(branch), systemPrompt)
	maxSteps := runMaxSteps()

	flog.Debug("[chat-agent] harness prompt session=%s model=%s workspace=%s branch_entries=%d max_steps=%d text_len=%d",
		req.SessionID, resolvedName, workspace.Root, len(branch), maxSteps, textLen)

	cfg := agent.DefaultConfig()
	cfg.ModelName = resolvedName
	cfg.MaxSteps = maxSteps

	return harness.New(harness.Options{
		AgentOptions: agent.Options{
			InitialState: agentCtx,
			Config:       cfg,
			Model:        llmModel,
			Registry:     registry,
		},
		Session:      agentSession,
		SystemPrompt: systemPrompt,
		ModelName:    resolvedName,
	}), nil
}

func runMaxSteps() int {
	maxSteps := config.App.ChatAgent.MaxSteps
	if maxSteps <= 0 {
		return 30
	}
	return maxSteps
}

func executeRun(ctx context.Context, h *harness.Harness, req RunRequest, start time.Time) (string, error) {
	stream, err := h.Prompt(ctx, agent.NewUserMessage(req.Text))
	if err != nil {
		if err == agent.ErrAborted {
			flog.Info("[chat-agent] harness busy session=%s duration=%s", req.SessionID, time.Since(start).Round(time.Millisecond))
			return "Agent is busy, please try again shortly.", nil
		}
		flog.Error(fmt.Errorf("[chat-agent] harness prompt session=%s: %w", req.SessionID, err))
		return "", err
	}

	result, err := stream.Await(ctx)
	if err != nil {
		return "", awaitRunError(req.SessionID, start, err)
	}
	if err := h.WaitIdle(ctx); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] wait persist session=%s: %w", req.SessionID, err))
		return "", err
	}
	if result.Err != nil {
		flog.Error(fmt.Errorf("[chat-agent] agent loop session=%s: %w", req.SessionID, result.Err))
		return "", result.Err
	}

	reply := extractAssistantReply(result.Messages)
	if reply == "" {
		flog.Warn("[chat-agent] empty assistant reply session=%s duration=%s messages=%d",
			req.SessionID, time.Since(start).Round(time.Millisecond), len(result.Messages))
		return "I could not produce a reply.", nil
	}

	flog.Debug("[chat-agent] harness finished session=%s reply_len=%d duration=%s",
		req.SessionID, len(reply), time.Since(start).Round(time.Millisecond))
	return reply, nil
}

func awaitRunError(sessionID string, start time.Time, err error) error {
	if errors.Is(err, context.Canceled) {
		flog.Info("[chat-agent] run cancelled session=%s duration=%s", sessionID, time.Since(start).Round(time.Millisecond))
		return fmt.Errorf("chat session ended")
	}
	flog.Error(fmt.Errorf("[chat-agent] stream await session=%s: %w", sessionID, err))
	return err
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
		text := textFromParts(assistant.Parts)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
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

// DefaultRunTimeout is the maximum duration for one assistant turn.
const DefaultRunTimeout = 10 * time.Minute

// NewModelForTest overrides model creation in unit tests.
var NewModelForTest = agentllm.NewModel
