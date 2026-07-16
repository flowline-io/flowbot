package chatagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/internal/store"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	sessionTitleMaxLen     = 60
	sessionTitleGenTimeout = 30 * time.Second
	sessionTitleMaxTokens  = 32
)

const sessionTitleSystemPrompt = `You generate short chat session titles. Output a single concise title (3-8 words) summarizing the conversation topic. No quotes, no trailing punctuation, one line only.`

type sessionTitleModelFunc func(context.Context, string) (llms.Model, string, error)

type sessionTitleLLMFunc func(context.Context, string, string, string, sessionTitleModelFunc) (string, error)

var (
	sessionTitleGenWG    sync.WaitGroup
	sessionTitleInflight sync.Map
	sessionTitleWriteMu  sync.Map
	sessionTitleLLMMu    sync.RWMutex

	// sessionTitleModel resolves the model for async title generation.
	// It is separate from NewModelForTest so harness test doubles are not consumed by title jobs.
	sessionTitleModel sessionTitleModelFunc = agentllm.GetOrCreateModel

	// generateSessionTitleLLM is overridden in unit tests.
	generateSessionTitleLLM sessionTitleLLMFunc = generateSessionTitleWithLLM
)

// WaitForSessionTitleGenerationForTest blocks until all in-flight async title generations finish.
func WaitForSessionTitleGenerationForTest() {
	sessionTitleGenWG.Wait()
}

// ResetSessionTitleGenerationForTest clears per-session title generation tracking.
func ResetSessionTitleGenerationForTest() {
	sessionTitleInflight = sync.Map{}
	sessionTitleWriteMu = sync.Map{}
}

func lockSessionTitleWrite(sessionID string) func() {
	raw, _ := sessionTitleWriteMu.LoadOrStore(sessionID, &sync.Mutex{})
	mu, ok := raw.(*sync.Mutex)
	if !ok {
		mu = &sync.Mutex{}
		sessionTitleWriteMu.Store(sessionID, mu)
	}
	mu.Lock()
	return mu.Unlock
}

// maybeGenerateSessionTitle starts an async LLM title generation when the session has no title yet.
// All store.Database access runs inside the WaitGroup so WaitForSessionTitleGenerationForTest
// covers the full critical section (avoids racing test cleanup that swaps the global adapter).
func maybeGenerateSessionTitle(sessionID, userText, reply string) {
	if _, loaded := sessionTitleInflight.LoadOrStore(sessionID, struct{}{}); loaded {
		return
	}
	sessionTitleLLMMu.RLock()
	llmGen := generateSessionTitleLLM
	modelResolver := sessionTitleModel
	sessionTitleLLMMu.RUnlock()
	sessionTitleGenWG.Go(func() {
		defer sessionTitleInflight.Delete(sessionID)

		if store.Database == nil {
			return
		}
		sess, err := store.Database.GetChatSession(context.Background(), sessionID)
		if err != nil {
			flog.Warn("[chat-agent] title generation skipped session=%s: %v", sessionID, err)
			return
		}
		if strings.TrimSpace(sess.Title) != "" {
			return
		}
		_, chatModel, _, _, err := agentLoopConfig()
		if err != nil {
			flog.Warn("[chat-agent] title generation skipped session=%s: %v", sessionID, err)
			return
		}
		generateSessionTitleAsync(sessionID, userText, reply, chatModel, modelResolver, llmGen)
	})
}

// DisableSessionTitleLLMForTest skips outbound title LLM calls until restore runs.
func DisableSessionTitleLLMForTest() (restore func()) {
	sessionTitleLLMMu.Lock()
	orig := generateSessionTitleLLM
	generateSessionTitleLLM = skipSessionTitleLLM
	sessionTitleLLMMu.Unlock()
	return func() {
		sessionTitleLLMMu.Lock()
		generateSessionTitleLLM = orig
		sessionTitleLLMMu.Unlock()
	}
}

func skipSessionTitleLLM(context.Context, string, string, string, sessionTitleModelFunc) (string, error) {
	return "", fmt.Errorf("session title llm skipped for test")
}

func generateSessionTitleAsync(
	sessionID, userText, reply, chatModel string,
	modelResolver sessionTitleModelFunc,
	llmGen sessionTitleLLMFunc,
) {
	ctx, cancel := context.WithTimeout(context.Background(), sessionTitleGenTimeout)
	defer cancel()

	title, err := llmGen(ctx, userText, reply, chatModel, modelResolver)
	if err != nil {
		flog.Warn("[chat-agent] title generation failed session=%s: %v", sessionID, err)
		title = fallbackSessionTitle(userText)
	}
	title = sanitizeTitle(title)
	if title == "" {
		title = fallbackSessionTitle(userText)
	}

	unlock := lockSessionTitleWrite(sessionID)
	defer unlock()

	if store.Database == nil {
		flog.Warn("[chat-agent] title write skipped session=%s: database unavailable", sessionID)
		return
	}
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		flog.Warn("[chat-agent] title write skipped session=%s: %v", sessionID, err)
		return
	}
	if strings.TrimSpace(sess.Title) != "" {
		return
	}
	if err := store.Database.UpdateChatSessionTitle(ctx, sessionID, title); err != nil {
		flog.Warn("[chat-agent] title write failed session=%s: %v", sessionID, err)
		return
	}
	flog.Debug("[chat-agent] session title set session=%s title=%q", sessionID, title)
}

func generateSessionTitleWithLLM(
	ctx context.Context,
	userText, reply, chatModel string,
	modelResolver sessionTitleModelFunc,
) (string, error) {
	model, resolvedName, err := modelResolver(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("chat agent model: %w", err)
	}
	prompt := buildSessionTitlePrompt(userText, reply)
	return agentllm.Complete(ctx, model, sessionTitleSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, resolvedName, sessionTitleMaxTokens)
}

func buildSessionTitlePrompt(userText, reply string) string {
	userText = strings.TrimSpace(userText)
	reply = strings.TrimSpace(reply)
	if len(reply) > 500 {
		reply = reply[:500] + "..."
	}
	if len(userText) > 500 {
		userText = userText[:500] + "..."
	}
	return fmt.Sprintf("User: %s\n\nAssistant: %s", userText, reply)
}

// sanitizeTitle normalizes an LLM title for storage and display.
func sanitizeTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	for strings.Contains(title, "  ") {
		title = strings.ReplaceAll(title, "  ", " ")
	}
	title = strings.Trim(title, `"'「」『』`)
	title = strings.TrimRight(title, ".,;:!?。，；：！？")
	if len(title) > sessionTitleMaxLen {
		title = strings.TrimSpace(title[:sessionTitleMaxLen])
	}
	return title
}

func fallbackSessionTitle(userText string) string {
	userText = strings.TrimSpace(userText)
	userText = strings.ReplaceAll(userText, "\n", " ")
	if len(userText) > sessionTitleMaxLen {
		return strings.TrimSpace(userText[:sessionTitleMaxLen-3]) + "..."
	}
	return userText
}

// LoadSessionTitle returns the persisted title for one chat session.
func LoadSessionTitle(ctx context.Context, sessionID string) string {
	if store.Database == nil {
		return ""
	}
	sess, err := store.Database.GetChatSession(ctx, sessionID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(sess.Title)
}
