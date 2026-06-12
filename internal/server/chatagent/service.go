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
	agentresult "github.com/flowline-io/flowbot/pkg/agent/result"
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
func (*Service) Run(ctx context.Context, req RunRequest, sink StreamSink) (string, error) {
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

	h, err := getOrCreateHarness(ctx, req, textLen)
	if err != nil {
		return "", err
	}

	return executeRun(ctx, h, req, start, sink)
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
	stream, err := h.Prompt(ctx, agent.NewUserMessage(req.Text))
	if err != nil {
		if errors.Is(err, agent.ErrAborted) || agentresult.IsCode(err, "busy") {
			flog.Info("[chat-agent] harness busy session=%s duration=%s", req.SessionID, time.Since(start).Round(time.Millisecond))
			return "Agent is busy, please try again shortly.", nil
		}
		flog.Error(fmt.Errorf("[chat-agent] harness prompt session=%s: %w", req.SessionID, err))
		return "", err
	}

	var waitCoalescer func()
	if sink != nil && stream != nil {
		waitCoalescer = startStreamCoalescer(ctx, stream.Events(), sink, streamUpdateInterval)
	}

	if err := h.WaitIdle(ctx); err != nil {
		if waitCoalescer != nil {
			waitCoalescer()
		}
		return "", awaitRunError(req.SessionID, start, err)
	}
	if waitCoalescer != nil {
		waitCoalescer()
	}

	result := h.LastRunResult()
	if result.Err != nil {
		flog.Error(fmt.Errorf("[chat-agent] agent loop session=%s: %w", req.SessionID, result.Err))
		return "", result.Err
	}

	reply := extractAssistantReply(result.Messages)
	if reply == "" {
		flog.Warn("[chat-agent] empty assistant reply session=%s duration=%s messages=%d",
			req.SessionID, time.Since(start).Round(time.Millisecond), len(result.Messages))
		reply = "I could not produce a reply."
	}

	if sink != nil {
		if err := sink.Flush(ctx, reply); err != nil {
			flog.Warn("[chat-agent] stream flush session=%s: %v", req.SessionID, err)
		}
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

// NewModelForTest overrides model creation in unit tests.
var NewModelForTest = agentllm.GetOrCreateModel
