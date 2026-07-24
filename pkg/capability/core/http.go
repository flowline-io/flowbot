package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	defaultHTTPTimeout = 30 * time.Second
	maxHTTPBodyBytes   = 1 << 20
)

type httpRequestInput struct {
	method  string
	parsed  *url.URL
	timeout time.Duration
	body    io.Reader
	headers map[string]any
}

func httpRequestInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	in, err := parseHTTPRequestInput(params)
	if err != nil {
		return nil, err
	}
	if err := assertURLAllowed(in.parsed); err != nil {
		return nil, err
	}

	reqBody := in.body
	if reqBody == nil {
		reqBody = http.NoBody
	}
	req, err := http.NewRequestWithContext(ctx, in.method, in.parsed.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, v := range in.headers {
		req.Header.Set(k, fmt.Sprint(v))
	}

	resp, err := newHTTPClient(in.timeout).Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			flog.Warn("http_request: close response body: %v", cerr)
		}
	}()

	return invokeResultFromHTTPResponse(resp)
}

func parseHTTPRequestInput(params map[string]any) (httpRequestInput, error) {
	rawURL, err := capability.RequiredString(params, "url")
	if err != nil {
		return httpRequestInput{}, err
	}
	method, _ := capability.StringParam(params, "method")
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return httpRequestInput{}, types.Errorf(types.ErrInvalidArgument, "invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return httpRequestInput{}, types.Errorf(types.ErrInvalidArgument, "url scheme must be http or https")
	}

	body, err := httpBodyReader(params["body"])
	if err != nil {
		return httpRequestInput{}, err
	}
	var headers map[string]any
	if raw, ok := params["headers"]; ok && raw != nil {
		h, ok := raw.(map[string]any)
		if !ok {
			return httpRequestInput{}, types.Errorf(types.ErrInvalidArgument, "headers must be a map")
		}
		headers = h
	}
	return httpRequestInput{
		method:  method,
		parsed:  parsed,
		timeout: httpTimeoutFromParams(params),
		body:    body,
		headers: headers,
	}, nil
}

func httpTimeoutFromParams(params map[string]any) time.Duration {
	timeout := defaultHTTPTimeout
	raw, ok := params["timeout_seconds"]
	if !ok {
		return timeout
	}
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return time.Duration(v) * time.Second
		}
	case int:
		if v > 0 {
			return time.Duration(v) * time.Second
		}
	}
	return timeout
}

func httpBodyReader(raw any) (io.Reader, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return strings.NewReader(string(v)), nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "body must be a string")
	}
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return assertURLAllowed(req.URL)
		},
	}
}

func invokeResultFromHTTPResponse(resp *http.Response) (*capability.InvokeResult, error) {
	limited := io.LimitReader(resp.Body, maxHTTPBodyBytes+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	truncated := false
	if len(bodyBytes) > maxHTTPBodyBytes {
		bodyBytes = bodyBytes[:maxHTTPBodyBytes]
		truncated = true
	}
	headerOut := map[string]any{}
	for k, vals := range resp.Header {
		if len(vals) == 1 {
			headerOut[k] = vals[0]
		} else {
			headerOut[k] = vals
		}
	}
	return &capability.InvokeResult{
		Data: map[string]any{
			"status":    resp.StatusCode,
			"headers":   headerOut,
			"body":      string(bodyBytes),
			"truncated": truncated,
		},
		Text: fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

func assertURLAllowed(u *url.URL) error {
	cfg := config.App.Core.HTTP
	host := u.Hostname()
	if host == "" {
		return types.Errorf(types.ErrForbidden, "empty host")
	}
	if hostAllowed(host, cfg.AllowHosts) {
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		if !cfg.AllowPrivate && isBlockedHostname(host) {
			return types.Errorf(types.ErrForbidden, "host %q is not allowed", host)
		}
		return fmt.Errorf("resolve host: %w", err)
	}
	if cfg.AllowPrivate {
		return nil
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return types.Errorf(types.ErrForbidden, "host %q resolves to blocked address %s", host, ip)
		}
	}
	if isBlockedHostname(host) {
		return types.Errorf(types.ErrForbidden, "host %q is not allowed", host)
	}
	return nil
}

func hostAllowed(host string, allow []string) bool {
	host = strings.ToLower(host)
	for _, a := range allow {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

func isBlockedHostname(host string) bool {
	h := strings.ToLower(host)
	switch h {
	case "localhost", "metadata.google.internal":
		return true
	}
	return strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local")
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}
