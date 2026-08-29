package llm

import (
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const defaultLLMHTTPTimeout = 10 * time.Minute

// openaiHTTPTransport builds the OpenAI-compatible HTTP transport chain.
// Idle detection is applied on chat-completion response bodies only; dial-level
// idle wrappers are avoided because they break HTTP keep-alive connection reuse.
func openaiHTTPTransport(withThinking bool) http.RoundTripper {
	var transport http.RoundTripper = cloneDefaultHTTPTransport()
	if withThinking {
		transport = &thinkingTransport{base: transport}
	}
	transport = &streamIdleTransport{base: transport}
	transport = &errorLogTransport{base: transport}
	return otelhttp.NewTransport(transport)
}

// openaiHTTPClient returns an HTTP client for OpenAI-compatible providers.
func openaiHTTPClient(withThinking bool) *http.Client {
	return &http.Client{
		Transport: openaiHTTPTransport(withThinking),
		Timeout:   llmHTTPTimeout(),
	}
}

func llmHTTPTimeout() time.Duration {
	timeout := config.App.ChatAgent.RunTimeout
	if timeout <= 0 {
		return defaultLLMHTTPTimeout
	}
	return timeout
}

func cloneDefaultHTTPTransport() *http.Transport {
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		return dt.Clone()
	}
	return &http.Transport{}
}

// errorLogTransport logs RoundTrip failures before langchaingo sanitizes net.Error
// into "network error: failed to reach API server". Logged URLs omit query and userinfo.
type errorLogTransport struct {
	base http.RoundTripper
}

func (t *errorLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	start := time.Now()
	resp, err := base.RoundTrip(req)
	if err != nil {
		method := ""
		var reqURL *url.URL
		if req != nil {
			method = req.Method
			reqURL = req.URL
		}
		flog.Warn("[agent-llm] http roundtrip failed method=%s url=%s duration=%s err=%v",
			method, redactHTTPURL(reqURL), time.Since(start).Round(time.Millisecond), err)
	}
	return resp, err
}

func redactHTTPURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	clone := *u
	clone.User = nil
	clone.RawQuery = ""
	clone.ForceQuery = false
	clone.Fragment = ""
	clone.RawFragment = ""
	return clone.String()
}
