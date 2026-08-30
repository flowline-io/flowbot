package llm

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func TestErrorLogTransportRestoresAPIErrorBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "400 param incorrect",
			status: http.StatusBadRequest,
			body:   `{"error":{"message":"Param Incorrect","param":"tool_calls.arguments"}}`,
		},
		{
			name:   "500 without param",
			status: http.StatusInternalServerError,
			body:   `{"error":{"message":"internal"}}`,
		},
		{
			name:   "404 plain json",
			status: http.StatusNotFound,
			body:   `{"error":{"message":"not found","param":"model"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transport := &errorLogTransport{base: statusRoundTripper{status: tt.status, body: tt.body}}
			req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/chat/completions", http.NoBody)
			require.NoError(t, err)
			resp, gotErr := transport.RoundTrip(req)
			require.NoError(t, gotErr)
			require.NotNil(t, resp)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.body, string(body))
		})
	}
}

func TestParseLLMAPIErrorExtractsMessageAndParam(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		body        string
		wantMessage string
		wantParam   string
	}{
		{
			name:        "miMo param incorrect",
			body:        `{"error":{"message":"Param Incorrect","param":"The reasoning_content in the thinking mode must be passed back to the API."}}`,
			wantMessage: "Param Incorrect",
			wantParam:   "The reasoning_content in the thinking mode must be passed back to the API.",
		},
		{
			name:        "missing param",
			body:        `{"error":{"message":"internal"}}`,
			wantMessage: "internal",
			wantParam:   "",
		},
		{
			name:        "not json",
			body:        `not-json`,
			wantMessage: "",
			wantParam:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			message, param := parseLLMAPIError([]byte(tt.body))
			assert.Equal(t, tt.wantMessage, message)
			assert.Equal(t, tt.wantParam, param)
		})
	}
}

type failingRoundTripper struct {
	err error
}

func (f failingRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, f.err
}

type statusRoundTripper struct {
	status int
	body   string
}

func (s statusRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}
