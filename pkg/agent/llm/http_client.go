package llm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const defaultLLMHTTPTimeout = 10 * time.Minute

// openaiHTTPTransport builds the OpenAI-compatible HTTP transport chain.
// Idle detection is applied on chat-completion response bodies only; dial-level
// idle wrappers are avoided because they break HTTP keep-alive connection reuse.
// Indexed tool-call SSE rewrite sits outside idle wrapping so inner reads still
// reset the idle timer.
func openaiHTTPTransport(withThinking bool) http.RoundTripper {
	var transport http.RoundTripper = cloneDefaultHTTPTransport()
	if withThinking {
		transport = &thinkingTransport{base: transport}
	}
	transport = &streamIdleTransport{base: transport}
	transport = &toolCallIndexTransport{base: transport}
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
		return resp, err
	}
	logLLMHTTPStatusError(req, resp)
	return resp, nil
}

const maxLLMErrorBody = 8 << 10

func logLLMHTTPStatusError(req *http.Request, resp *http.Response) {
	if resp == nil || resp.StatusCode < http.StatusBadRequest || resp.Body == nil {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLLMErrorBody))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return
	}
	message, param := parseLLMAPIError(body)
	reqURL := ""
	if req != nil {
		reqURL = redactHTTPURL(req.URL)
	}
	flog.Warn("[agent-llm] http error status=%d url=%s message=%s param=%s",
		resp.StatusCode, reqURL, message, param)
}

type llmAPIErrorBody struct {
	Error llmAPIErrorFields `json:"error"`
}

type llmAPIErrorFields struct {
	Message string `json:"message"`
	Param   any    `json:"param"`
}

func parseLLMAPIError(body []byte) (message, param string) {
	var payload llmAPIErrorBody
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return "", ""
	}
	message = strings.TrimSpace(payload.Error.Message)
	if payload.Error.Param != nil {
		param = strings.TrimSpace(fmt.Sprint(payload.Error.Param))
	}
	return message, param
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
