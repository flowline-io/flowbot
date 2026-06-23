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

	if _, ok := payload["thinking"]; !ok {
		payload["thinking"] = map[string]string{"type": "enabled"}
	}
	if _, ok := payload["reasoning_effort"]; !ok {
		payload["reasoning_effort"] = "high"
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
	return &http.Client{Transport: &deepSeekThinkingTransport{base: http.DefaultTransport}}
}

// DeepSeekThinkingHTTPClientForTest exposes the DeepSeek thinking HTTP client for tests.
func DeepSeekThinkingHTTPClientForTest(base http.RoundTripper) *http.Client {
	return &http.Client{Transport: &deepSeekThinkingTransport{base: base}}
}
