package llm_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripRecorder struct {
	req *http.Request
}

func (r *roundTripRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req.Clone(req.Context())
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
		r.req.Body = io.NopCloser(bytes.NewReader(body))
		r.req.ContentLength = int64(len(body))
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Header:     make(http.Header),
	}, nil
}

func TestDeepSeekThinkingHTTPClientInjectsThinking(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantInject bool
		wantEffort string
	}{
		{
			name:       "chat completion gets thinking fields",
			path:       "/v1/chat/completions",
			body:       `{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"hi"}]}`,
			wantInject: true,
			wantEffort: "high",
		},
		{
			name:       "non chat path unchanged",
			path:       "/v1/models",
			body:       `{"model":"deepseek-v4-flash"}`,
			wantInject: false,
		},
		{
			name:       "preserves existing thinking fields",
			path:       "/v1/chat/completions",
			body:       `{"model":"deepseek-v4-flash","reasoning_effort":"max","thinking":{"type":"enabled"}}`,
			wantInject: true,
			wantEffort: "max",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := &roundTripRecorder{}
			client := llm.DeepSeekThinkingHTTPClientForTest(rec)
			req, err := http.NewRequest(http.MethodPost, "https://api.deepseek.com"+tt.path, strings.NewReader(tt.body))
			require.NoError(t, err)

			_, err = client.Do(req)
			require.NoError(t, err)
			require.NotNil(t, rec.req)

			payload, err := io.ReadAll(rec.req.Body)
			require.NoError(t, err)

			var parsed map[string]any
			require.NoError(t, sonic.Unmarshal(payload, &parsed))
			if !tt.wantInject {
				_, hasThinking := parsed["thinking"]
				assert.False(t, hasThinking)
				return
			}
			assert.Equal(t, tt.wantEffort, parsed["reasoning_effort"])
			thinking, ok := parsed["thinking"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "enabled", thinking["type"])
			}
		})
	}
}
