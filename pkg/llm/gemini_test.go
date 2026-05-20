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

func TestGemini_Generate_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		wantContent  string
	}{
		{
			name:         "happy path simple response",
			responseBody: `{"candidates":[{"content":{"parts":[{"text":"hello"}],"role":"model"},"finishReason":"STOP"}]}`,
			statusCode:   http.StatusOK,
			wantContent:  "hello",
		},
		{
			name:         "happy path multi part response",
			responseBody: `{"candidates":[{"content":{"parts":[{"text":"This is"},{"text":" a test"}],"role":"model"},"finishReason":"STOP"}]}`,
			statusCode:   http.StatusOK,
			wantContent:  "This is a test",
		},
		{
			name:         "happy path empty text",
			responseBody: `{"candidates":[{"content":{"parts":[{"text":""}],"role":"model"},"finishReason":"STOP"}]}`,
			statusCode:   http.StatusOK,
			wantContent:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "generateContent")
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprint(w, tt.responseBody)
			}))
			t.Cleanup(srv.Close)

			p, err := llm.NewProvider(config.Model{
				Provider: llm.ProviderGemini,
				ApiKey:   "test-gemini-key",
				BaseUrl:  srv.URL,
			})
			require.NoError(t, err)

			msg, err := p.Generate(context.Background(), &llm.GenerateRequest{
				Model:    "gemini-3.1-pro",
				Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, msg.Content)
		})
	}
}

func TestGemini_Generate_EmptyCandidates(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"candidates":[]}`)
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderGemini,
		ApiKey:   "test-gemini-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	msg, err := p.Generate(context.Background(), &llm.GenerateRequest{
		Model:    "gemini-3.1-pro",
		Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
	})
	require.NoError(t, err)
	assert.Empty(t, msg.Content)
}

func TestGemini_Generate_APIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "400 bad request", statusCode: http.StatusBadRequest, body: `{"error":{"code":400,"message":"invalid request"}}`},
		{name: "500 server error", statusCode: http.StatusInternalServerError, body: `{"error":{"code":500,"message":"internal error"}}`},
		{name: "429 rate limited", statusCode: http.StatusTooManyRequests, body: `{"error":{"code":429,"message":"resource exhausted"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			t.Cleanup(srv.Close)

			p, err := llm.NewProvider(config.Model{
				Provider: llm.ProviderGemini,
				ApiKey:   "test-gemini-key",
				BaseUrl:  srv.URL,
			})
			require.NoError(t, err)

			_, err = p.Generate(context.Background(), &llm.GenerateRequest{
				Model:    "gemini-3.1-pro",
				Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
			})
			require.Error(t, err)
		})
	}
}

func TestGemini_GenerateStream_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "streamGenerateContent")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"}}]}`)
		flusher.Flush()
		_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":" world"}],"role":"model"}}]}`)
		flusher.Flush()
		_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"STOP"}]}`)
		flusher.Flush()
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderGemini,
		ApiKey:   "test-gemini-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	ch, err := p.GenerateStream(context.Background(), &llm.GenerateRequest{
		Model:    "gemini-3.1-pro",
		Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
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

func TestGemini_GenerateStream_CtxCancel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":"first"}],"role":"model"}}]}`)
		flusher.Flush()
	}))
	t.Cleanup(srv.Close)

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderGemini,
		ApiKey:   "test-gemini-key",
		BaseUrl:  srv.URL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.GenerateStream(ctx, &llm.GenerateRequest{
		Model:    "gemini-3.1-pro",
		Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
	})
	require.NoError(t, err)

	f := <-ch
	assert.Equal(t, "first", f.Content)
	cancel()
}
