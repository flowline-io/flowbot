package chatagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

const (
	knowledgeMetadataGenTimeout      = 30 * time.Second
	knowledgeMetadataMaxTokens       = 256
	knowledgeMetadataContentMaxBytes = 12 * 1024
	knowledgeMetadataTitleMaxLen     = 256
	knowledgeMetadataSummaryMaxLen   = 1024
	knowledgeMetadataMinTags         = 3
	knowledgeMetadataMaxTags         = 6
	knowledgeMetadataTagMaxLen       = 32
)

const knowledgeMetadataSystemPrompt = `You generate metadata for a markdown knowledge document used in search.
Match the language of the document content (not the path).
Return ONLY a single JSON object with keys:
- "title": concise document title (required, one line)
- "tags": array of 3-6 short search tags (lowercase words or short phrases, each under 32 characters)
- "summary": one-line summary for search results (required)
No markdown fences, no commentary.`

// KnowledgeMetadata is AI-generated title, tags, and summary for a knowledge document.
type KnowledgeMetadata struct {
	Title   string
	Tags    []string
	Summary string
}

type knowledgeMetadataModelFunc func(context.Context, string) (llms.Model, string, error)

type knowledgeMetadataLLMFunc func(context.Context, string, string, string, knowledgeMetadataModelFunc) (KnowledgeMetadata, error)

var (
	knowledgeMetadataLLMMu sync.RWMutex

	knowledgeMetadataModel knowledgeMetadataModelFunc = agentllm.GetOrCreateModel

	generateKnowledgeMetadataLLM knowledgeMetadataLLMFunc = generateKnowledgeMetadataWithLLM
)

// GenerateKnowledgeMetadata derives title, tags, and summary from path and content via chat_model.
// It does not persist; the caller fills a draft form.
func GenerateKnowledgeMetadata(ctx context.Context, path, content string) (KnowledgeMetadata, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return KnowledgeMetadata{}, fmt.Errorf("content is required")
	}
	chatModel := config.ChatAgentChatModel()
	if chatModel == "" {
		return KnowledgeMetadata{}, fmt.Errorf("chat agent not configured")
	}
	knowledgeMetadataLLMMu.RLock()
	llmGen := generateKnowledgeMetadataLLM
	modelResolver := knowledgeMetadataModel
	knowledgeMetadataLLMMu.RUnlock()

	genCtx, cancel := context.WithTimeout(ctx, knowledgeMetadataGenTimeout)
	defer cancel()
	return llmGen(genCtx, path, content, chatModel, modelResolver)
}

// SetKnowledgeMetadataLLMForTest replaces the LLM generator until restore runs.
func SetKnowledgeMetadataLLMForTest(fn knowledgeMetadataLLMFunc) (restore func()) {
	knowledgeMetadataLLMMu.Lock()
	orig := generateKnowledgeMetadataLLM
	generateKnowledgeMetadataLLM = fn
	knowledgeMetadataLLMMu.Unlock()
	return func() {
		knowledgeMetadataLLMMu.Lock()
		generateKnowledgeMetadataLLM = orig
		knowledgeMetadataLLMMu.Unlock()
	}
}

// SetKnowledgeMetadataGeneratorForTest overrides end-to-end metadata generation for tests outside this package.
func SetKnowledgeMetadataGeneratorForTest(fn func(context.Context, string, string) (KnowledgeMetadata, error)) (restore func()) {
	return SetKnowledgeMetadataLLMForTest(func(ctx context.Context, path, content, _ string, _ knowledgeMetadataModelFunc) (KnowledgeMetadata, error) {
		return fn(ctx, path, content)
	})
}

func generateKnowledgeMetadataWithLLM(
	ctx context.Context,
	path, content, chatModel string,
	modelResolver knowledgeMetadataModelFunc,
) (KnowledgeMetadata, error) {
	model, resolvedName, err := modelResolver(ctx, chatModel)
	if err != nil {
		return KnowledgeMetadata{}, fmt.Errorf("chat agent model: %w", err)
	}
	prompt := buildKnowledgeMetadataPrompt(path, content)
	raw, err := agentllm.Complete(ctx, model, knowledgeMetadataSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, resolvedName, knowledgeMetadataMaxTokens)
	if err != nil {
		return KnowledgeMetadata{}, err
	}
	meta, err := parseKnowledgeMetadataJSON(raw)
	if err != nil {
		return KnowledgeMetadata{}, err
	}
	meta = sanitizeKnowledgeMetadata(meta)
	if meta.Title == "" || meta.Summary == "" || len(meta.Tags) < knowledgeMetadataMinTags {
		return KnowledgeMetadata{}, fmt.Errorf("incomplete metadata from model")
	}
	return meta, nil
}

func buildKnowledgeMetadataPrompt(path, content string) string {
	path = strings.TrimSpace(path)
	content = truncateKnowledgeMetadataContent(content)
	if path == "" {
		path = "(none)"
	}
	return fmt.Sprintf("Path: %s\n\nContent:\n%s", path, content)
}

func truncateKnowledgeMetadataContent(content string) string {
	if len(content) <= knowledgeMetadataContentMaxBytes {
		return content
	}
	truncated := content[:knowledgeMetadataContentMaxBytes]
	if !utf8.ValidString(truncated) {
		for len(truncated) > 0 && !utf8.ValidString(truncated) {
			truncated = truncated[:len(truncated)-1]
		}
	}
	return truncated + "\n\n[truncated]"
}

func parseKnowledgeMetadataJSON(raw string) (KnowledgeMetadata, error) {
	raw = strings.TrimSpace(raw)
	raw = stripMarkdownJSONFence(raw)
	if idx := strings.Index(raw, "{"); idx >= 0 {
		if end := strings.LastIndex(raw, "}"); end > idx {
			raw = raw[idx : end+1]
		}
	}
	var payload struct {
		Title   string   `json:"title"`
		Tags    []string `json:"tags"`
		Summary string   `json:"summary"`
	}
	if err := sonic.UnmarshalString(raw, &payload); err != nil {
		return KnowledgeMetadata{}, fmt.Errorf("parse metadata json: %w", err)
	}
	return KnowledgeMetadata{
		Title:   payload.Title,
		Tags:    payload.Tags,
		Summary: payload.Summary,
	}, nil
}

func stripMarkdownJSONFence(raw string) string {
	if !strings.HasPrefix(raw, "```") {
		return raw
	}
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "json") {
		raw = strings.TrimSpace(raw[4:])
	}
	if idx := strings.LastIndex(raw, "```"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw)
}

func sanitizeKnowledgeMetadata(meta KnowledgeMetadata) KnowledgeMetadata {
	meta.Title = clampOneLine(meta.Title, knowledgeMetadataTitleMaxLen)
	meta.Summary = clampOneLine(meta.Summary, knowledgeMetadataSummaryMaxLen)
	meta.Tags = normalizeKnowledgeMetadataTags(meta.Tags)
	return meta
}

func clampOneLine(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	if utf8.RuneCountInString(s) > maxLen {
		runes := []rune(s)
		s = strings.TrimSpace(string(runes[:maxLen]))
	}
	return s
}

func normalizeKnowledgeMetadataTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = clampOneLine(tag, knowledgeMetadataTagMaxLen)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
		if len(out) >= knowledgeMetadataMaxTags {
			break
		}
	}
	return out
}
