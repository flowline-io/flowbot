// Package client provides a Go client SDK for the Flowbot server's web service API.
//
// The client supports typed access to all major bot webservice endpoints including
// kanban, bookmark, user, search, dev, and server APIs.
//
// Usage:
//
//	c := client.NewClient("http://localhost:6060", "your-access-token")
//
//	// List kanban tasks
//	tasks, err := c.Kanban.List(ctx, 1, kanboard.Active)
//
//	// Create a bookmark
//	bookmark, err := c.Bookmark.Create(ctx, "https://example.com")
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"resty.dev/v3"
)

// Client is the main client for the Flowbot API.
type Client struct {
	baseURL           string
	rc                *resty.Client
	debugErrorHookSet bool

	// Resource clients
	Kanban   *KanbanClient
	Bookmark *BookmarkClient
	Reader   *ReaderClient
	User     *UserClient
	Search   *SearchClient
	Dev      *DevClient
	Server   *ServerClient
	Hub      *HubClient
}

// NewClient creates a new client with the given server URL and access token.
// The token is sent as the X-AccessToken header for authentication.
func NewClient(serverURL, token string) *Client {
	rc := resty.New()
	rc.SetBaseURL(serverURL)
	rc.SetHeader("X-AccessToken", token)
	rc.SetHeader("Content-Type", "application/json")
	rc.SetTimeout(30 * time.Second)

	c := &Client{
		baseURL: serverURL,
		rc:      rc,
	}

	// Initialize resource clients
	c.Kanban = &KanbanClient{c: c}
	c.Bookmark = &BookmarkClient{c: c}
	c.Reader = &ReaderClient{c: c}
	c.User = &UserClient{c: c}
	c.Search = &SearchClient{c: c}
	c.Dev = &DevClient{c: c}
	c.Server = &ServerClient{c: c}
	c.Hub = &HubClient{c: c}

	return c
}

// SetTimeout sets the request timeout for the client.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.rc.SetTimeout(timeout)
}

// SetDebug enables or disables debug mode for the underlying HTTP client.
// Debug mode prints full HTTP request and response details to stderr.
// An OnError hook is also registered to print request info on connection failures.
func (c *Client) SetDebug(debug bool) {
	c.rc.SetDebug(debug)
	if debug && !c.debugErrorHookSet {
		c.debugErrorHookSet = true
		c.rc.OnError(func(req *resty.Request, err error) {
			fmt.Fprintf(os.Stderr, "\n==============================================================================\n")
			fmt.Fprintf(os.Stderr, "~~~ REQUEST (FAILED) ~~~\n")
			fmt.Fprintf(os.Stderr, "%s  %s\n", req.Method, req.URL)
			fmt.Fprintf(os.Stderr, "ERROR  : %v\n", err)
			fmt.Fprintf(os.Stderr, "==============================================================================\n")
		})
	}
}

// Get performs a GET request and unmarshals the response data into result.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	resp, err := c.rc.R().SetContext(ctx).Get(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Post performs a POST request with the given body and unmarshals the response data into result.
func (c *Client) Post(ctx context.Context, path string, body any, result any) error {
	resp, err := c.rc.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Patch performs a PATCH request with the given body and unmarshals the response data into result.
func (c *Client) Patch(ctx context.Context, path string, body any, result any) error {
	resp, err := c.rc.R().SetContext(ctx).SetBody(body).Patch(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Put performs a PUT request with the given body and unmarshals the response data into result.
func (c *Client) Put(ctx context.Context, path string, body any, result any) error {
	resp, err := c.rc.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// Delete performs a DELETE request with the given body and unmarshals the response data into result.
func (c *Client) Delete(ctx context.Context, path string, body any, result any) error {
	resp, err := c.rc.R().SetContext(ctx).SetBody(body).Delete(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseResponse(resp, result)
}

// RawRequest returns a resty request builder for custom requests.
// Use this when you need full control over the request.
func (c *Client) RawRequest() *resty.Request {
	return c.rc.R()
}

// parseResponse unmarshals the HTTP response into a protocol.Response,
// checks for errors, and unmarshals the Data field into the target result.
func parseResponse(resp *resty.Response, result any) error {
	body := resp.Bytes()
	if len(body) == 0 {
		return &APIError{
			StatusCode: resp.StatusCode(),
			Message:    "empty response",
		}
	}

	var r protocol.Response
	if err := json.Unmarshal(body, &r); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode(),
			Message:    fmt.Sprintf("parse response: %v", err),
		}
	}

	if r.Status != protocol.Success {
		msg := r.Message
		if msg == "" {
			msg = "unknown error"
		}
		return &APIError{
			StatusCode: resp.StatusCode(),
			RetCode:    r.RetCode,
			Message:    msg,
		}
	}

	if result != nil && r.Data != nil {
		data, err := json.Marshal(r.Data)
		if err != nil {
			return fmt.Errorf("marshal response data: %w", err)
		}
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
	}

	return nil
}

// APIError represents an error returned by the Flowbot API.
type APIError struct {
	StatusCode int
	RetCode    string
	Message    string
}

func (e *APIError) Error() string {
	if e.RetCode != "" {
		return fmt.Sprintf("API error (code %s): %s", e.RetCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.RetCode == "10009"
	}
	return false
}

// IsUnauthorized returns true if the error indicates unauthorized access.
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.RetCode == "60005"
	}
	return false
}

// stringOr returns the string value for key from kv, or defaultVal if not found or not a string.
func stringOr(kv map[string]any, key string, defaultVal string) string {
	if v, ok := kv[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}
