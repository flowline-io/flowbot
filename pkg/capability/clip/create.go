// Package clip provides the clip capability for creating shareable markdown clips.
package clip

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	slugLen            = 8
	slugAlphabet       = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	metaMaxTokens      = 128
	metaGenTimeout     = 30 * time.Second
	titleMaxLen        = 80
	descriptionMaxLen  = 200
	contentPromptLimit = 4000
	slugMaxAttempts    = 8
	// MaxContentBytes is the maximum accepted markdown body size for create.
	MaxContentBytes = 512 * 1024
)

const metaSystemPrompt = `You summarize markdown clips for sharing previews.
Respond with a single JSON object only, no markdown fences:
{"title":"...","description":"..."}
title: concise, 3-12 words, no trailing punctuation.
description: one or two sentences, under 160 characters, plain text.`

// Persister stores and loads clips.
type Persister interface {
	// CreateClip inserts a clip row keyed by slug.
	CreateClip(ctx context.Context, slug, title, description, content, createdBy string) error
	// GetClipBySlug returns a clip by slug, or nil when not found.
	GetClipBySlug(ctx context.Context, slug string) (*Record, error)
}

// Record is a persisted clip returned by get operations.
type Record struct {
	// Slug is the public short identifier.
	Slug string
	// Title is the clip title.
	Title string
	// Description is the short preview text.
	Description string
	// Content is the markdown body.
	Content string
	// CreatedBy is the optional creator identifier.
	CreatedBy string
	// CreatedAt is when the clip was created.
	CreatedAt time.Time
}

// Meta holds LLM-generated (or fallback) title and description.
type Meta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type metaModelFunc func(context.Context, string) (llms.Model, string, error)

type metaLLMFunc func(context.Context, string, metaModelFunc) (Meta, error)

// serviceMarker is a non-nil instance used for hub registration.
type serviceMarker struct{}

var (
	persisterMu sync.RWMutex
	persister   Persister

	metaLLMMu    sync.RWMutex
	metaModel    metaModelFunc = agentllm.GetOrCreateModel
	generateMeta metaLLMFunc   = generateMetaWithLLM
)

// SetPersister wires the persistence backend used by clip.create.
func SetPersister(p Persister) {
	persisterMu.Lock()
	defer persisterMu.Unlock()
	persister = p
}

func getPersister() Persister {
	persisterMu.RLock()
	defer persisterMu.RUnlock()
	return persister
}

// SetMetaLLMForTest overrides LLM metadata generation in unit tests.
func SetMetaLLMForTest(fn metaLLMFunc) {
	metaLLMMu.Lock()
	defer metaLLMMu.Unlock()
	if fn == nil {
		generateMeta = generateMetaWithLLM
		return
	}
	generateMeta = fn
}

// Register registers the clip capability with hub and invoker registry.
func Register() error {
	return capability.Register(capability.Spec{
		Type:        hub.CapClip,
		Description: "Create shareable markdown clips with LLM-generated title and description",
		Instance:    serviceMarker{},
		Ops: []capability.OpDef{
			{
				Name: OpCreate, Description: "Create a markdown clip and return its public URL", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "content", Type: "string", Required: true, Description: "Markdown body"},
					{Name: "created_by", Type: "string", Required: false, Description: "Optional creator identifier"},
				},
				Handler: createInvoker,
			},
			{
				Name: OpGet, Description: "Get a markdown clip by slug",
				Input: []hub.ParamDef{
					{Name: "slug", Type: "string", Required: true, Description: "Clip slug"},
				},
				Handler: getInvoker,
			},
			{
				Name: OpHealth, Description: "Health check",
				Handler: healthInvoker,
			},
		},
	})
}

func createInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	content, err := capability.RequiredString(params, "content")
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "content is required")
	}
	if len(content) > MaxContentBytes {
		return nil, types.Errorf(types.ErrInvalidArgument, "content exceeds %d bytes", MaxContentBytes)
	}
	createdBy, _ := capability.StringParam(params, "created_by")

	p := getPersister()
	if p == nil {
		return nil, types.Errorf(types.ErrUnavailable, "clip persister is not configured")
	}

	metaLLMMu.RLock()
	metaFn := generateMeta
	modelFn := metaModel
	metaLLMMu.RUnlock()

	meta, err := metaFn(ctx, content, modelFn)
	if err != nil {
		flog.Warn("clip meta llm failed, using fallback: %v", err)
		meta = fallbackMeta(content)
	}
	meta.Title = sanitizeLine(meta.Title, titleMaxLen)
	meta.Description = sanitizeLine(meta.Description, descriptionMaxLen)
	if meta.Title == "" || meta.Description == "" {
		fb := fallbackMeta(content)
		if meta.Title == "" {
			meta.Title = fb.Title
		}
		if meta.Description == "" {
			meta.Description = fb.Description
		}
	}

	var slug string
	for range slugMaxAttempts {
		slug, err = newSlug(slugLen)
		if err != nil {
			return nil, types.Errorf(types.ErrInternal, "generate slug: %v", err)
		}
		err = p.CreateClip(ctx, slug, meta.Title, meta.Description, content, createdBy)
		if err == nil {
			url := "/c/" + slug
			return &capability.InvokeResult{
				Data: map[string]any{
					"slug":        slug,
					"title":       meta.Title,
					"description": meta.Description,
					"url":         url,
				},
				Text: fmt.Sprintf("clip created: %s", url),
			}, nil
		}
		if !isUniqueViolation(err) {
			return nil, fmt.Errorf("create clip: %w", err)
		}
	}
	return nil, types.Errorf(types.ErrInternal, "create clip: slug collision after %d attempts", slugMaxAttempts)
}

func getInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	slug, err := capability.RequiredString(params, "slug")
	if err != nil {
		return nil, err
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "slug is required")
	}

	p := getPersister()
	if p == nil {
		return nil, types.Errorf(types.ErrUnavailable, "clip persister is not configured")
	}

	row, err := p.GetClipBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get clip: %w", err)
	}
	if row == nil {
		return nil, types.Errorf(types.ErrNotFound, "clip %q not found", slug)
	}

	url := "/c/" + row.Slug
	return &capability.InvokeResult{
		Data: map[string]any{
			"slug":        row.Slug,
			"title":       row.Title,
			"description": row.Description,
			"content":     row.Content,
			"created_by":  row.CreatedBy,
			"created_at":  row.CreatedAt.UTC().Format(time.RFC3339),
			"url":         url,
		},
		Text: fmt.Sprintf("clip %s: %s", row.Slug, row.Title),
	}, nil
}

func healthInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	ready := getPersister() != nil
	status := "ok"
	if !ready {
		status = "unavailable"
	}
	return &capability.InvokeResult{
		Data: map[string]any{"status": status, "ready": ready},
		Text: status,
	}, nil
}

func generateMetaWithLLM(ctx context.Context, content string, modelResolver metaModelFunc) (Meta, error) {
	chatModel := config.ChatAgentChatModel()
	if chatModel == "" {
		return Meta{}, fmt.Errorf("chat agent model is not configured")
	}
	model, resolvedName, err := modelResolver(ctx, chatModel)
	if err != nil {
		return Meta{}, fmt.Errorf("clip meta model: %w", err)
	}
	genCtx, cancel := context.WithTimeout(ctx, metaGenTimeout)
	defer cancel()
	prompt := truncateRunes(strings.TrimSpace(content), contentPromptLimit)
	raw, err := agentllm.Complete(genCtx, model, metaSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, resolvedName, metaMaxTokens)
	if err != nil {
		return Meta{}, err
	}
	return parseMetaJSON(raw)
}

func parseMetaJSON(raw string) (Meta, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var meta Meta
	if err := sonic.Unmarshal([]byte(raw), &meta); err != nil {
		return Meta{}, fmt.Errorf("parse meta json: %w", err)
	}
	return meta, nil
}

func fallbackMeta(content string) Meta {
	title := firstHeading(content)
	if title == "" {
		title = sanitizeLine(firstLine(content), titleMaxLen)
	}
	if title == "" {
		title = "Untitled clip"
	}
	desc := sanitizeLine(plainPreview(content), descriptionMaxLen)
	if desc == "" {
		desc = title
	}
	return Meta{Title: title, Description: desc}
}

func firstHeading(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimLeft(trimmed, "#")
			return sanitizeLine(trimmed, titleMaxLen)
		}
	}
	return ""
}

func firstLine(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed
	}
	return ""
}

func plainPreview(content string) string {
	var b strings.Builder
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "```") {
			continue
		}
		trimmed = strings.TrimLeft(trimmed, "-*+ ")
		if trimmed == "" {
			continue
		}
		if b.Len() > 0 {
			_ = b.WriteByte(' ')
		}
		_, _ = b.WriteString(trimmed)
		if b.Len() >= descriptionMaxLen {
			break
		}
	}
	return b.String()
}

func sanitizeLine(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.Trim(s, `"'「」『』`)
	s = strings.TrimRight(s, ".,;:!?。，；：！？")
	return truncateRunes(s, maxLen)
}

func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxLen]))
}

func newSlug(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("invalid slug length")
	}
	alphabetLen := big.NewInt(int64(len(slugAlphabet)))
	buf := make([]byte, n)
	for i := range n {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		buf[i] = slugAlphabet[idx.Int64()]
	}
	return string(buf), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "constraint")
}

// WordCount returns a rough word count for markdown display metadata.
func WordCount(content string) int {
	count := 0
	inWord := false
	for _, r := range content {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			count++
			inWord = true
		}
	}
	return count
}
