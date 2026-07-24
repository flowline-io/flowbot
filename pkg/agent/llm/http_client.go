package llm

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/flowline-io/flowbot/pkg/config"
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
