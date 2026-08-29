package llm

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMHTTPTimeout(t *testing.T) {
	prev := config.App.ChatAgent.RunTimeout
	t.Cleanup(func() { config.App.ChatAgent.RunTimeout = prev })

	tests := []struct {
		name       string
		runTimeout time.Duration
		want       time.Duration
	}{
		{
			name:       "default when unset",
			runTimeout: 0,
			want:       defaultLLMHTTPTimeout,
		},
		{
			name:       "uses configured run timeout",
			runTimeout: 5 * time.Minute,
			want:       5 * time.Minute,
		},
		{
			name:       "custom short timeout",
			runTimeout: 30 * time.Second,
			want:       30 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.RunTimeout = tt.runTimeout
			assert.Equal(t, tt.want, llmHTTPTimeout())
		})
	}
}

func TestRedactHTTPURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "strips query and userinfo",
			raw:  "https://key:secret@api.example.com/v1/chat/completions?api_key=leak",
			want: "https://api.example.com/v1/chat/completions",
		},
		{
			name: "keeps host and path",
			raw:  "https://api.deepseek.com/chat/completions",
			want: "https://api.deepseek.com/chat/completions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, redactHTTPURL(u))
		})
	}
	assert.Empty(t, redactHTTPURL(nil))
}

func TestErrorLogTransportPassesThrough(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("dial tcp: i/o timeout")
	transport := &errorLogTransport{base: failingRoundTripper{err: wantErr}}
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/chat/completions?token=secret", http.NoBody)
	require.NoError(t, err)
	resp, gotErr := transport.RoundTrip(req)
	assert.Nil(t, resp)
	assert.Equal(t, wantErr, gotErr)
}

type failingRoundTripper struct {
	err error
}

func (f failingRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, f.err
}
