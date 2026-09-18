package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestMergeRecognizedSpans_DropsLanguageMismatch(t *testing.T) {
	t.Parallel()

	spans := []recognizedSpan{
		{EntityType: "EMAIL_ADDRESS", Start: 0, End: 5, Score: 0.9, Language: "en", snippetHint: "a@b.c"},
		{EntityType: "PERSON", Start: 10, End: 13, Score: 0.8, Language: "en", snippetHint: "张三"},
		{EntityType: "PERSON", Start: 20, End: 24, Score: 0.7, Language: "zh", snippetHint: "John"},
		{EntityType: "PHONE_NUMBER", Start: 30, End: 41, Score: 0.9, Language: "zh", snippetHint: "13812345678"},
	}
	got := mergeRecognizedSpans(spans)
	require.Len(t, got, 2)
	assert.Equal(t, "EMAIL_ADDRESS", got[0].EntityType)
	assert.Equal(t, "PHONE_NUMBER", got[1].EntityType)
}

func TestPIITable_StablePlaceholdersAndRestore(t *testing.T) {
	t.Parallel()

	tab := newPIITable()
	text := "mail me at alice@example.com please"
	spans := []recognizedSpan{{EntityType: "EMAIL_ADDRESS", Start: 11, End: 28, Score: 0.99}}
	tab.mu.Lock()
	anon := tab.anonymizeLocked(text, spans)
	tab.mu.Unlock()
	assert.Contains(t, anon, "{{PII_EMAIL_ADDRESS_1}}")
	assert.NotContains(t, anon, "alice@example.com")

	tab.mu.Lock()
	anon2 := tab.anonymizeLocked(text, spans)
	restored := tab.restoreLocked(anon)
	tab.mu.Unlock()
	assert.Equal(t, anon, anon2)
	assert.Equal(t, text, restored)
}

func TestPIITable_HashCacheSkipsReanalyzeShape(t *testing.T) {
	t.Parallel()

	tab := newPIITable()
	text := "call 13812345678 now"
	spans := []recognizedSpan{{EntityType: "PHONE_NUMBER", Start: 5, End: 16, Score: 0.9}}
	tab.mu.Lock()
	first := tab.anonymizeLocked(text, spans)
	second := tab.anonymizeLocked(text, nil) // empty spans still hits cache
	tab.mu.Unlock()
	assert.Equal(t, first, second)
	assert.Contains(t, first, "{{PII_PHONE_NUMBER_1}}")
}

func TestStreamRestorer_SplitsPlaceholder(t *testing.T) {
	t.Parallel()

	tab := newPIITable()
	tab.mu.Lock()
	_ = tab.allocateLocked("EMAIL_ADDRESS", "a@b.com")
	tab.mu.Unlock()

	r := newStreamRestorer(tab)
	part1 := r.push("hello {{PII_EMAIL_")
	assert.Equal(t, "hello ", part1)
	part2 := r.push("ADDRESS_1}} world")
	assert.Equal(t, "a@b.com world", part2)
	assert.Empty(t, r.flush())
}

func TestPIIModel_DisabledPassthrough(t *testing.T) {
	prev := config.App.PII
	t.Cleanup(func() { config.App.PII = prev })
	config.App.PII = config.PIIConfig{Enabled: false}

	fake := NewFakeModel(ResponseScript{Content: "ok"})
	model := wrapPIIModel(fake)
	resp, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "alice@example.com"),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Choices[0].Content)
	assert.Equal(t, 1, fake.Calls())
	msgs := fake.LastMessages()
	require.Len(t, msgs, 1)
	text, ok := msgs[0].Parts[0].(llms.TextContent)
	require.True(t, ok)
	assert.Equal(t, "alice@example.com", text.Text)
}

func TestPIIModel_FailClosedOnAnalyzerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	t.Cleanup(srv.Close)

	prev := config.App.PII
	t.Cleanup(func() { config.App.PII = prev })
	config.App.PII = config.PIIConfig{
		Enabled:        true,
		AnalyzerURL:    srv.URL,
		Timeout:        prev.Timeout,
		ScoreThreshold: 0.5,
		SessionTTL:     prev.SessionTTL,
	}
	config.App.Normalize()

	fake := NewFakeModel(ResponseScript{Content: "should-not-run"})
	model := wrapPIIModel(fake)
	_, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "alice@example.com"),
	})
	require.Error(t, err)
	assert.Equal(t, 0, fake.Calls())
}

func TestPIIModel_AnonymizeAndRestoreWithSession(t *testing.T) {
	var analyzeCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyze" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		analyzeCalls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var req analyzeRequest
		if err := sonic.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var items []analyzeResponseItem
		if strings.Contains(req.Text, "alice@example.com") {
			start := strings.Index(req.Text, "alice@example.com")
			items = append(items, analyzeResponseItem{
				EntityType: "EMAIL_ADDRESS",
				Start:      start,
				End:        start + len("alice@example.com"),
				Score:      0.99,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		raw, err := sonic.Marshal(items)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)

	prev := config.App.PII
	t.Cleanup(func() { config.App.PII = prev })
	config.App.PII = config.PIIConfig{
		Enabled:        true,
		AnalyzerURL:    srv.URL,
		ScoreThreshold: 0.5,
	}
	config.App.Normalize()

	fake := NewFakeModel(ResponseScript{
		Chunks: []string{"reply ", "{{PII_EMAIL_", "ADDRESS_1}}", " done"},
		ToolCalls: []llms.ToolCall{{
			ID:   "c1",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "send",
				Arguments: `{"to":"{{PII_EMAIL_ADDRESS_1}}"}`,
			},
		}},
	})
	model := wrapPIIModel(fake)
	ctx := WithPIISession(context.Background(), "sess-pii-1")

	var streamed strings.Builder
	resp, err := model.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "system keeps alice@example.com"),
		llms.TextParts(llms.ChatMessageTypeHuman, "email alice@example.com please"),
	}, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		_, _ = streamed.Write(chunk)
		return nil
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.GreaterOrEqual(t, analyzeCalls.Load(), int32(2)) // en + zh
	msgs := fake.LastMessages()
	require.Len(t, msgs, 2)
	sys, ok := msgs[0].Parts[0].(llms.TextContent)
	require.True(t, ok)
	assert.Contains(t, sys.Text, "alice@example.com") // system skipped
	human, ok := msgs[1].Parts[0].(llms.TextContent)
	require.True(t, ok)
	assert.Contains(t, human.Text, "{{PII_EMAIL_ADDRESS_1}}")
	assert.NotContains(t, human.Text, "alice@example.com")

	assert.Contains(t, streamed.String(), "alice@example.com")
	assert.NotContains(t, streamed.String(), "{{PII_EMAIL_ADDRESS_1}}")
	assert.Contains(t, resp.Choices[0].Content, "alice@example.com")
	require.NotNil(t, resp.Choices[0].ToolCalls[0].FunctionCall)
	assert.Contains(t, resp.Choices[0].ToolCalls[0].FunctionCall.Arguments, "alice@example.com")

	// Second turn reuses session table / hash cache — still stable ID.
	fake2 := NewFakeModel(ResponseScript{Content: "ok"})
	model2 := wrapPIIModel(fake2)
	_, err = model2.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "email alice@example.com please"),
	})
	require.NoError(t, err)
	human2, ok := fake2.LastMessages()[0].Parts[0].(llms.TextContent)
	require.True(t, ok)
	assert.Equal(t, human.Text, human2.Text)
}

func TestPIIModel_EnabledHotToggle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyze" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, "[]")
	}))
	t.Cleanup(srv.Close)

	prev := config.App.PII
	t.Cleanup(func() { config.App.PII = prev })
	config.App.PII = config.PIIConfig{Enabled: false, AnalyzerURL: srv.URL}
	config.App.Normalize()

	fake := NewFakeModel(
		ResponseScript{Content: "one"},
		ResponseScript{Content: "two"},
	)
	model := wrapPIIModel(fake)

	_, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "hi"),
	})
	require.NoError(t, err)

	config.App.PII.Enabled = true
	_, err = model.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "hi"),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, fake.Calls())
}

func TestWithPIISession_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := WithPIISession(context.Background(), " abc ")
	assert.Equal(t, "abc", PIISessionFromContext(ctx))
	assert.Empty(t, PIISessionFromContext(context.Background()))
}

func TestChunkText(t *testing.T) {
	t.Parallel()
	input := strings.Repeat("a", 100) + "\n" + strings.Repeat("b", 100)
	chunks := chunkText(input, 80)
	require.GreaterOrEqual(t, len(chunks), 2)
	rebuilt := ""
	for _, c := range chunks {
		assert.Equal(t, len(rebuilt), c.offset)
		rebuilt += c.text
	}
	assert.Equal(t, input, rebuilt)
}

func TestRuneRangeToBytes_CJK(t *testing.T) {
	t.Parallel()
	// "姓名张三手机" — Presidio reports rune indices.
	text := "姓名张三手机"
	// "张三" is runes 2..4
	start, end, ok := runeRangeToBytes(text, 2, 4)
	require.True(t, ok)
	assert.Equal(t, "张三", text[start:end])

	spans := spansFromAnalyzerItems(text, "zh", []analyzeResponseItem{{
		EntityType: "PERSON", Start: 2, End: 4, Score: 0.9,
	}})
	require.Len(t, spans, 1)
	assert.Equal(t, "张三", text[spans[0].Start:spans[0].End])

	tab := newPIITable()
	tab.mu.Lock()
	anon := tab.anonymizeLocked(text, spans)
	tab.mu.Unlock()
	assert.Contains(t, anon, "{{PII_PERSON_1}}")
	assert.NotContains(t, anon, "张三")
}

func TestPIIModel_FlushTrailingPlaceholderToStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "[]")
	}))
	t.Cleanup(srv.Close)

	prev := config.App.PII
	t.Cleanup(func() { config.App.PII = prev })
	config.App.PII = config.PIIConfig{Enabled: true, AnalyzerURL: srv.URL}
	config.App.Normalize()

	tab := newPIITable()
	tab.mu.Lock()
	_ = tab.allocateLocked("EMAIL_ADDRESS", "a@b.com")
	tab.mu.Unlock()
	globalPIISessions.mu.Lock()
	globalPIISessions.tables["flush-sess"] = tab
	globalPIISessions.mu.Unlock()
	t.Cleanup(func() {
		globalPIISessions.mu.Lock()
		delete(globalPIISessions.tables, "flush-sess")
		globalPIISessions.mu.Unlock()
	})

	// Last chunk ends mid-placeholder; flush must still emit the held suffix to the stream.
	fake := NewFakeModel(ResponseScript{
		Chunks: []string{"see {{PII_EMAIL_ADDRESS_1"},
	})
	model := wrapPIIModel(fake)
	ctx := WithPIISession(context.Background(), "flush-sess")

	var streamed strings.Builder
	_, err := model.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "hi"),
	}, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		_, _ = streamed.Write(chunk)
		return nil
	}))
	require.NoError(t, err)
	// "see " emitted during push; incomplete token flushed after GenerateContent.
	assert.Equal(t, "see {{PII_EMAIL_ADDRESS_1", streamed.String())
}

func TestChunkText_DoesNotSplitUTF8Rune(t *testing.T) {
	t.Parallel()
	// Three-byte runes; force a mid-rune candidate cut.
	input := strings.Repeat("中", 20)
	chunks := chunkText(input, 10)
	require.NotEmpty(t, chunks)
	rebuilt := ""
	for _, c := range chunks {
		assert.True(t, utf8.ValidString(c.text))
		rebuilt += c.text
	}
	assert.Equal(t, input, rebuilt)
}
