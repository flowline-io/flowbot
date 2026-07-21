package llm

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
)

// deepSeekThinkingTransport injects DeepSeek V4 thinking parameters into chat completion requests.
type deepSeekThinkingTransport struct {
	base http.RoundTripper
}

func (t *deepSeekThinkingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if req.Body == nil || !strings.Contains(req.URL.Path, "chat/completions") {
		return base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	var payload map[string]any
	if err := sonic.Unmarshal(body, &payload); err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return base.RoundTrip(req)
	}

	level := ThinkingLevelFromContext(req.Context())
	enabled := deepSeekThinkingEnabled(level)
	if _, ok := payload["thinking"]; !ok {
		if enabled {
			payload["thinking"] = map[string]string{"type": "enabled"}
		} else {
			payload["thinking"] = map[string]string{"type": "disabled"}
		}
	}
	// DeepSeek rejects reasoning_effort=none; omit the field when thinking is off.
	if enabled {
		if _, ok := payload["reasoning_effort"]; !ok {
			payload["reasoning_effort"] = deepSeekReasoningEffort(level)
		}
	} else {
		delete(payload, "reasoning_effort")
	}

	patched, err := sonic.Marshal(payload)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return base.RoundTrip(req)
	}

	req.Body = io.NopCloser(bytes.NewReader(patched))
	req.ContentLength = int64(len(patched))
	return base.RoundTrip(req)
}

func deepSeekThinkingHTTPClient() *http.Client {
	return openaiHTTPClient(true)
}

// DeepSeekThinkingHTTPClientForTest exposes the DeepSeek thinking HTTP client for tests.
func DeepSeekThinkingHTTPClientForTest(base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}
	return &http.Client{
		Transport: &streamIdleTransport{base: &deepSeekThinkingTransport{base: base}},
		Timeout:   llmHTTPTimeout(),
	}
}
