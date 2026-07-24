package chatagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/internal/store"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	sessionSummaryScopeDefault   = "default"
	sessionSummaryMaxInputChars  = 24000
	sessionSummaryMaxTokens      = 512
	sessionSummaryGenTimeout     = 90 * time.Second
	sessionSummaryStaleClaimAge  = 10 * time.Minute
	sessionSummaryWorkerInterval = 30 * time.Second
)

const sessionSummarySystemPrompt = `You summarize archived chat sessions for later retrieval.
Write a concise paragraph (or short bullets) covering the main topics, decisions, and outcomes.
Do not invent details. Prefer concrete nouns over vague language.`

type sessionSummaryLLMFunc func(context.Context, string, string, sessionTitleModelFunc) (string, error)

var (
	sessionSummaryGenWG    sync.WaitGroup
	sessionSummaryLLMMu    sync.RWMutex
	sessionSummaryModel    sessionTitleModelFunc = agentllm.GetOrCreateModel
	generateSessionSummary sessionSummaryLLMFunc = generateSessionSummaryWithLLM

	sessionSummaryWorkerOnce sync.Once
	sessionSummaryWorkerStop chan struct{}
)

// SetSessionArchived updates the archive flag and enqueues summary generation when archiving.
func SetSessionArchived(ctx context.Context, sessionID string, archived bool) error {
	db := store.Database
	if db == nil {
		return fmt.Errorf("chatagent: database unavailable")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("chatagent: empty session id")
	}
	if err := db.UpdateChatSessionArchived(ctx, sessionID, archived); err != nil {
		return err
	}
	if archived {
		OnSessionArchived(ctx, sessionID)
	}
	return nil
}

// OnSessionArchived upserts a pending summary and kicks async processing.
// Callers that already set archived=true may invoke this without SetSessionArchived.
func OnSessionArchived(ctx context.Context, sessionID string) {
	if err := enqueueSessionSummary(ctx, sessionID); err != nil {
		flog.Warn("[chat-agent] session summary pending upsert session=%s: %v", sessionID, err)
	}
}

// RetrySessionSummary requeues a failed or pending summary for generation.
func RetrySessionSummary(ctx context.Context, sessionID string) error {
	return enqueueSessionSummary(ctx, sessionID)
}

func enqueueSessionSummary(ctx context.Context, sessionID string) error {
	// Clone so callers that pass Fiber/fasthttp buffer aliases stay safe after the request ends.
	sessionID = strings.Clone(strings.TrimSpace(sessionID))
	if sessionID == "" {
		return fmt.Errorf("chatagent: empty session id")
	}
	db := store.Database
	if db == nil {
		return fmt.Errorf("chatagent: database unavailable")
	}
	title := ""
	if sess, err := db.GetChatSession(ctx, sessionID); err == nil && sess != nil {
		title = strings.TrimSpace(sess.Title)
	}
	if _, err := db.UpsertAgentSessionSummaryPending(ctx, sessionID, sessionSummaryScopeDefault, title); err != nil {
		return err
	}
	kickSessionSummaryWorker()
	return nil
}

// StartSessionSummaryWorker starts the background claim/process loop once.
// When ctx is cancelled the worker stops.
func StartSessionSummaryWorker(ctx context.Context) {
	sessionSummaryWorkerOnce.Do(func() {
		sessionSummaryWorkerStop = make(chan struct{})
		go runSessionSummaryWorker(sessionSummaryWorkerStop)
		flog.Info("[chat-agent] session summary worker started")
		go func() {
			select {
			case <-ctx.Done():
				StopSessionSummaryWorkerForTest()
			case <-sessionSummaryWorkerStop:
			}
		}()
	})
}

// StopSessionSummaryWorkerForTest stops the worker loop for tests.
func StopSessionSummaryWorkerForTest() {
	if sessionSummaryWorkerStop != nil {
		select {
		case <-sessionSummaryWorkerStop:
		default:
			close(sessionSummaryWorkerStop)
		}
	}
}

// WaitForSessionSummaryGenerationForTest waits for in-flight summary jobs.
func WaitForSessionSummaryGenerationForTest() {
	sessionSummaryGenWG.Wait()
}

func kickSessionSummaryWorker() {
	sessionSummaryGenWG.Go(func() {
		processOneSessionSummary(context.Background())
	})
}

func runSessionSummaryWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(sessionSummaryWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if store.Database == nil {
				continue
			}
			if _, err := store.Database.RequeueStaleAgentSessionSummaryPending(context.Background(), sessionSummaryStaleClaimAge); err != nil {
				flog.Warn("[chat-agent] requeue stale session summaries: %v", err)
			}
			for range 5 {
				if !processOneSessionSummary(context.Background()) {
					break
				}
			}
		}
	}
}

func processOneSessionSummary(ctx context.Context) bool {
	db := store.Database
	if db == nil {
		return false
	}
	claimToken := uuid.NewString()
	row, err := db.ClaimAgentSessionSummaryPending(ctx, claimToken)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return false
		}
		flog.Warn("[chat-agent] claim session summary: %v", err)
		return false
	}

	title, summary, genErr := generateSessionSummaryForSession(ctx, row.SessionFlag, row.Title)
	if genErr != nil {
		flog.Warn("[chat-agent] session summary failed session=%s: %v", row.SessionFlag, genErr)
		if markErr := db.MarkAgentSessionSummaryFailed(ctx, row.SessionFlag, claimToken, genErr.Error()); markErr != nil {
			flog.Warn("[chat-agent] mark session summary failed session=%s: %v", row.SessionFlag, markErr)
		}
		return true
	}
	if err := db.MarkAgentSessionSummaryReady(ctx, row.SessionFlag, claimToken, title, summary); err != nil {
		flog.Warn("[chat-agent] mark session summary ready session=%s: %v", row.SessionFlag, err)
	}
	return true
}

func generateSessionSummaryForSession(ctx context.Context, sessionID, existingTitle string) (title, summary string, err error) {
	input, err := buildSessionSummaryInput(ctx, sessionID)
	if err != nil {
		return "", "", err
	}
	chatModel := "gpt-test"
	if _, m, _, _, cfgErr := agentLoopConfig(); cfgErr == nil && m != "" {
		chatModel = m
	}
	sessionSummaryLLMMu.RLock()
	llmGen := generateSessionSummary
	modelResolver := sessionSummaryModel
	sessionSummaryLLMMu.RUnlock()

	genCtx, cancel := context.WithTimeout(ctx, sessionSummaryGenTimeout)
	defer cancel()
	summary, err = llmGen(genCtx, input, chatModel, modelResolver)
	if err != nil {
		return "", "", err
	}
	title = strings.TrimSpace(existingTitle)
	if title == "" {
		title = fallbackSessionTitle(input)
	}
	return title, strings.TrimSpace(summary), nil
}

func generateSessionSummaryWithLLM(ctx context.Context, input, chatModel string, modelResolver sessionTitleModelFunc) (string, error) {
	model, resolvedName, err := modelResolver(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("chat agent model: %w", err)
	}
	return agentllm.Complete(ctx, model, sessionSummarySystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}, resolvedName, sessionSummaryMaxTokens)
}

func buildSessionSummaryInput(ctx context.Context, sessionID string) (string, error) {
	storage := NewDBStorage(sessionID, types.Uid(""), "")
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return "", err
	}
	built := session.BuildContext(branch)
	var b strings.Builder
	for _, m := range built.Messages {
		line := formatSummaryMessage(m)
		if line == "" {
			continue
		}
		if b.Len()+len(line)+1 > sessionSummaryMaxInputChars {
			break
		}
		_, _ = b.WriteString(line)
		_ = b.WriteByte('\n')
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("session has no summarizable messages")
	}
	return text, nil
}

func formatSummaryMessage(m msg.AgentMessage) string {
	role := string(m.Role())
	text := strings.TrimSpace(agentMessagePlainText(m))
	if text == "" {
		return ""
	}
	if len(text) > 2000 {
		text = text[:2000] + "..."
	}
	if role == "" {
		role = "message"
	}
	return role + ": " + text
}

func agentMessagePlainText(m msg.AgentMessage) string {
	switch v := m.(type) {
	case msg.UserMessage:
		return contentPartsText(v.Parts)
	case msg.AssistantMessage:
		return msg.AssistantDisplayText(v)
	case msg.CompactionSummaryMessage:
		return v.Summary
	case msg.BranchSummaryMessage:
		return v.Summary
	case msg.ToolResultMessage:
		return contentPartsText(v.Parts)
	default:
		return ""
	}
}

func contentPartsText(parts []msg.ContentPart) string {
	var b strings.Builder
	for _, part := range parts {
		if tp, ok := part.(msg.TextPart); ok {
			if b.Len() > 0 {
				_ = b.WriteByte('\n')
			}
			_, _ = b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// DisableSessionSummaryLLMForTest skips outbound summary LLM calls until restore runs.
func DisableSessionSummaryLLMForTest() (restore func()) {
	sessionSummaryLLMMu.Lock()
	orig := generateSessionSummary
	generateSessionSummary = func(context.Context, string, string, sessionTitleModelFunc) (string, error) {
		return "", fmt.Errorf("session summary llm skipped for test")
	}
	sessionSummaryLLMMu.Unlock()
	return func() {
		sessionSummaryLLMMu.Lock()
		generateSessionSummary = orig
		sessionSummaryLLMMu.Unlock()
	}
}

// SetSessionSummaryLLMForTest overrides the summary LLM function.
func SetSessionSummaryLLMForTest(fn sessionSummaryLLMFunc) (restore func()) {
	sessionSummaryLLMMu.Lock()
	orig := generateSessionSummary
	generateSessionSummary = fn
	sessionSummaryLLMMu.Unlock()
	return func() {
		sessionSummaryLLMMu.Lock()
		generateSessionSummary = orig
		sessionSummaryLLMMu.Unlock()
	}
}
