package chatagent

import (
	"context"
	"fmt"
	"strings"
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

// generateSessionTitleLLM is overridden in unit tests.
var generateSessionTitleLLM = generateSessionTitleWithLLM

// maybeGenerateSessionTitle starts an async LLM title generation when the session has no title yet.
func maybeGenerateSessionTitle(sessionID, userText, reply string) {
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
	go generateSessionTitleAsync(sessionID, userText, reply)
}

func generateSessionTitleAsync(sessionID, userText, reply string) {
	ctx, cancel := context.WithTimeout(context.Background(), sessionTitleGenTimeout)
	defer cancel()

	title, err := generateSessionTitleLLM(ctx, userText, reply)
	if err != nil {
		flog.Warn("[chat-agent] title generation failed session=%s: %v", sessionID, err)
		title = fallbackSessionTitle(userText)
	}
	title = sanitizeTitle(title)
	if title == "" {
		title = fallbackSessionTitle(userText)
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

func generateSessionTitleWithLLM(ctx context.Context, userText, reply string) (string, error) {
	_, chatModel, _, _, err := agentLoopConfig()
	if err != nil {
		return "", err
	}
	model, resolvedName, err := NewModelForTest(ctx, chatModel)
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
