package chatagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

// RunRequest carries one user turn for the chat assistant.
type RunRequest struct {
	SessionID string
	Text      string
}

// Service orchestrates chat assistant agent runs for direct chat sessions.
type Service struct {
	mu        sync.Mutex
	sessions  map[string]*sync.Mutex
	workspace coding.Workspace
}

// NewService creates a chat agent service using current application config.
func NewService() *Service {
	return &Service{
		sessions:  make(map[string]*sync.Mutex),
		workspace: WorkspaceFromConfig(),
	}
}

// Run executes one agent turn and returns the assistant reply text.
func (s *Service) Run(ctx context.Context, req RunRequest) (string, error) {
	if !agentllm.AgentEnabled(agentName) {
		return "", fmt.Errorf("chat agent is disabled or model is not configured")
	}
	if strings.TrimSpace(req.Text) == "" {
		return "", fmt.Errorf("empty message")
	}

	lock := s.sessionLock(req.SessionID)
	lock.Lock()
	defer lock.Unlock()

	modelName := agentllm.AgentModelName(agentName)
	llmModel, resolvedName, err := NewModelForTest(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("chat agent model: %w", err)
	}

	registry, err := NewRegistry(s.workspace)
	if err != nil {
		return "", err
	}

	store := session.New(NewDBStorage(req.SessionID))
	branch, err := store.GetBranch(ctx, "")
	if err != nil {
		return "", fmt.Errorf("load session branch: %w", err)
	}
	built := session.BuildContext(branch)
	agentCtx := session.ToAgentContext(built, SystemPrompt(s.workspace))

	cfg := agent.DefaultConfig()
	cfg.ModelName = resolvedName
	cfg.MaxSteps = 30

	h := harness.New(harness.Options{
		AgentOptions: agent.Options{
			InitialState: agentCtx,
			Config:       cfg,
			Model:        llmModel,
			Registry:     registry,
		},
		Session:      store,
		SystemPrompt: SystemPrompt(s.workspace),
		ModelName:    resolvedName,
	})

	stream, err := h.Prompt(ctx, agent.NewUserMessage(req.Text))
	if err != nil {
		if err == agent.ErrAborted {
			return "Agent is busy, please try again shortly.", nil
		}
		return "", err
	}

	result, err := stream.Await(ctx)
	if err != nil {
		return "", err
	}
	if result.Err != nil {
		return "", result.Err
	}

	reply := extractAssistantReply(result.Messages)
	if reply == "" {
		return "I could not produce a reply.", nil
	}
	return reply, nil
}

func (s *Service) sessionLock(sessionID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lock, ok := s.sessions[sessionID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	s.sessions[sessionID] = lock
	return lock
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
