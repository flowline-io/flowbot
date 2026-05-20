package llm_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestAnthropic_Generate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		wantContent  string
	}{
		{
			name:         "happy path simple response",
			responseBody: `{"content":[{"type":"text","text":"hello"}],"role":"assistant"}`,
			statusCode:   http.StatusOK,
			wantContent:  "hello",
		},
		{
			name:         "happy path multi block response",
			responseBody: `{"content":[{"type":"text","text":"first block."},{"type":"text","text":" second block."}],"role":"assistant"}`,
			statusCode:   http.StatusOK,
			wantContent:  "first block. second block.",
		},
		{
			name:         "happy path empty text",
			responseBody: `{"content":[{"type":"text","text":""}],"role":"assistant"}`,
			statusCode:   http.StatusOK,
			wantContent:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/messages", r.URL.Path)
				assert.Equal(t, "test-anthropic-key", r.Header.Get("x-api-key"))
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprint(w, tt.responseBody)
			}))
			t.Cleanup(srv.Close)

			p, err := llm.NewProvider(config.Model{
				Provider: llm.ProviderAnthropic,
				ApiKey:   "test-anthropic-key",
				BaseUrl:  srv.URL,
			})
			require.NoError(t, err)

			msg, err := p.Generate(context.Background(), &llm.GenerateRequest{
				Model:    "claude-opus-4.7",
				Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, msg.Content)
		})
	}
}

func TestAnthropic_Generate_EmptyContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"content":[],"role":"assistant"}`)
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderAnthropic,
		ApiKey:   "test-anthropic-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	msg, err := p.Generate(context.Background(), &llm.GenerateRequest{
		Model:    "claude-opus-4.7",
		Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
	})
	require.NoError(t, err)
	assert.Empty(t, msg.Content)
}

func TestAnthropic_Generate_APIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "401 unauthorized", statusCode: http.StatusUnauthorized, body: `{"error":{"type":"authentication_error","message":"invalid api key"}}`},
		{name: "429 rate limited", statusCode: http.StatusTooManyRequests, body: `{"error":{"type":"rate_limit_error","message":"rate limit exceeded"}}`},
		{name: "500 server error", statusCode: http.StatusInternalServerError, body: `{"error":{"type":"server_error","message":"internal error"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			t.Cleanup(srv.Close)

			p, err := llm.NewProvider(config.Model{
				Provider: llm.ProviderAnthropic,
				ApiKey:   "test-anthropic-key",
				BaseUrl:  srv.URL,
			})
			require.NoError(t, err)

			_, err = p.Generate(context.Background(), &llm.GenerateRequest{
				Model:    "claude-opus-4.7",
				Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
			})
			require.Error(t, err)
		})
	}
}

func TestAnthropic_GenerateStream_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`)
		flusher.Flush()
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":" world"}}`)
		flusher.Flush()
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		flusher.Flush()
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderAnthropic,
		ApiKey:   "test-anthropic-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	ch, err := p.GenerateStream(context.Background(), &llm.GenerateRequest{
		Model:     "claude-opus-4.7",
		Messages:  []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
		MaxTokens: 1024,
	})
	require.NoError(t, err)

	var contents []string
	var done bool
	for f := range ch {
		require.NoError(t, f.Err)
		if f.Content != "" {
			contents = append(contents, f.Content)
		}
		if f.Done {
			done = true
		}
	}
	assert.True(t, done)
	assert.Equal(t, []string{"Hello", " world"}, contents)
}

func TestAnthropic_GenerateStream_CtxCancel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"first"}}`)
		flusher.Flush()
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderAnthropic,
		ApiKey:   "test-anthropic-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.GenerateStream(ctx, &llm.GenerateRequest{
		Model:     "claude-opus-4.7",
		Messages:  []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
		MaxTokens: 1024,
	})
	require.NoError(t, err)

	f := <-ch
	assert.Equal(t, "first", f.Content)
	cancel()
}
