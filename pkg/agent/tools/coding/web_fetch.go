package coding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// WebFetchTool downloads text content from an http(s) URL.
type WebFetchTool struct {
	HTTPClient *http.Client
	MaxOutput  int
	// AllowLoopback permits localhost/loopback hosts (intended for tests).
	AllowLoopback bool
}

// Name returns the tool identifier.
func (WebFetchTool) Name() string { return "web_fetch" }

// Description explains the tool to the model.
func (WebFetchTool) Description() string {
	return "Fetches text content from an http(s) URL; blocks localhost/loopback/link-local hosts including redirects; response truncated for context safety"
}

// Parameters returns the JSON schema for tool arguments.
func (WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Absolute http or https URL to fetch",
			},
		},
		"required": []string{"url"},
	}
}

// Execute fetches the URL body.
func (t WebFetchTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	parsed, errResult := t.parseFetchURL(id, args)
	if errResult != nil {
		return *errResult, nil
	}
	if onUpdate != nil {
		_ = onUpdate("fetching...")
	}
	body, status, contentType, errResult := t.doFetch(ctx, id, parsed)
	if errResult != nil {
		return *errResult, nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: formatFetchOutput(parsed.String(), status, contentType, body, t.MaxOutput)}},
	}, nil
}

func (t WebFetchTool) parseFetchURL(id string, args map[string]any) (*url.URL, *msg.ToolResultMessage) {
	rawURL := strings.TrimSpace(fmt.Sprint(args["url"]))
	if rawURL == "" || rawURL == "<nil>" {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", "url is required", "provide an http(s) URL")
		return nil, &res
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", "invalid url", "provide a valid absolute URL")
		return nil, &res
	}
	if err := validateFetchURL(parsed, t.AllowLoopback); err != nil {
		res := tool.ErrorResult(id, t.Name(), "invalid_args", err.Error(), "use a public http(s) URL")
		return nil, &res
	}
	return parsed, nil
}

func (t WebFetchTool) doFetch(ctx context.Context, id string, parsed *url.URL) ([]byte, int, string, *msg.ToolResultMessage) {
	client := t.securedHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), http.NoBody)
	if err != nil {
		res := toolError(id, t.Name(), err.Error())
		return nil, 0, "", &res
	}
	resp, err := client.Do(req)
	if err != nil {
		res := toolError(id, t.Name(), fmt.Sprintf("fetch request: %v", err))
		return nil, 0, "", &res
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxFetchBytes)+1))
	if err != nil {
		res := toolError(id, t.Name(), fmt.Sprintf("read response: %v", err))
		return nil, 0, "", &res
	}
	if len(body) > MaxFetchBytes {
		body = body[:MaxFetchBytes]
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		res := toolError(id, t.Name(), fmt.Sprintf("fetch status %d", resp.StatusCode))
		return nil, 0, "", &res
	}
	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func (t WebFetchTool) securedHTTPClient() *http.Client {
	base := t.HTTPClient
	if base == nil {
		base = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	client := *base
	client.CheckRedirect = RedirectChecker(t.AllowLoopback, client.CheckRedirect)
	return &client
}

// RedirectChecker returns an http.Client CheckRedirect that re-validates each redirect target.
func RedirectChecker(allowLoopback bool, prev func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if err := validateFetchURL(req.URL, allowLoopback); err != nil {
			return fmt.Errorf("blocked redirect: %w", err)
		}
		if prev != nil {
			return prev(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
}

func formatFetchOutput(rawURL string, status int, contentType string, body []byte, maxOutput int) string {
	text := string(body)
	if len(body) >= MaxFetchBytes {
		text += "\n...(body truncated)"
	}
	out := fmt.Sprintf("URL: %s\nStatus: %d\nContent-Type: %s\n\n%s", rawURL, status, contentType, text)
	limit := maxOutput
	if limit <= 0 {
		limit = DefaultMaxOutput
	}
	if len(out) > limit {
		return out[:limit] + "\n...(output truncated)"
	}
	return out
}

func validateFetchURL(u *url.URL, allowLoopback bool) error {
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http and https URLs are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("url host is required")
	}
	if allowLoopback {
		return nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("localhost and loopback hosts are blocked")
	}
	if ip := net.ParseIP(host); ip != nil {
		return validateFetchIP(ip)
	}
	return nil
}

func validateFetchIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("loopback and link-local addresses are blocked")
	}
	return nil
}
