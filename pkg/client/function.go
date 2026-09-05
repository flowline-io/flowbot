package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// FunctionClient provides access to the named-functions management and call API.
type FunctionClient struct {
	c *Client
}

// FunctionInfo is a published function list entry.
type FunctionInfo struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Status  string `json:"status"`
}

// FunctionListResult contains published functions.
type FunctionListResult struct {
	Functions []FunctionInfo `json:"functions"`
}

// FunctionApplyResult is returned after applying a function bundle.
type FunctionApplyResult struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
	Status  string `json:"status"`
}

// FunctionExportResult holds an exported function snapshot.
type FunctionExportResult struct {
	Name       string `json:"name"`
	Version    int    `json:"version"`
	Metadata   string `json:"metadata"`
	Entrypoint string `json:"entrypoint"`
	Source     string `json:"source"`
}

// FunctionRunInfo is a single function run history entry.
type FunctionRunInfo struct {
	ID           int64  `json:"id"`
	FunctionName string `json:"function_name"`
	Version      int    `json:"version"`
	Status       string `json:"status"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	Error        string `json:"error,omitempty"`
	ResultJSON   string `json:"result_json,omitempty"`
}

// FunctionRunsResult contains run history for a function.
type FunctionRunsResult struct {
	Runs []FunctionRunInfo `json:"runs"`
}

// FunctionCallResult is returned from POST /call.
type FunctionCallResult struct {
	Name     string `json:"name"`
	Version  int    `json:"version"`
	RunID    int64  `json:"run_id"`
	Status   string `json:"status"`
	Result   any    `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Replayed bool   `json:"replayed,omitempty"`
}

// Apply upserts a function definition from metadata, entrypoint, and source (draft + publish).
func (f *FunctionClient) Apply(ctx context.Context, metadata, entrypoint, source string) (*FunctionApplyResult, error) {
	var result FunctionApplyResult
	err := f.c.Post(ctx, "/service/automate/functions/apply", map[string]string{
		"metadata":   metadata,
		"entrypoint": entrypoint,
		"source":     source,
	}, &result)
	return &result, err
}

// List returns published function definitions.
func (f *FunctionClient) List(ctx context.Context) (*FunctionListResult, error) {
	var result FunctionListResult
	err := f.c.Get(ctx, "/service/automate/functions/list", &result)
	return &result, err
}

// Get returns function metadata by name.
func (f *FunctionClient) Get(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	path := "/service/automate/functions/get/" + url.PathEscape(name)
	err := f.c.Get(ctx, path, &result)
	return result, err
}

// Export returns the published function snapshot including secrets and source.
func (f *FunctionClient) Export(ctx context.Context, name string) (*FunctionExportResult, error) {
	var result FunctionExportResult
	path := "/service/automate/functions/export/" + url.PathEscape(name)
	err := f.c.Get(ctx, path, &result)
	return &result, err
}

// Delete removes a function definition by name.
func (f *FunctionClient) Delete(ctx context.Context, name string) error {
	path := "/service/automate/functions/delete/" + url.PathEscape(name)
	var result map[string]any
	return f.c.Delete(ctx, path, nil, &result)
}

// Runs returns run history for a function.
func (f *FunctionClient) Runs(ctx context.Context, name string) (*FunctionRunsResult, error) {
	var result FunctionRunsResult
	path := "/service/automate/functions/runs/" + url.PathEscape(name)
	err := f.c.Get(ctx, path, &result)
	return &result, err
}

// Call invokes a published function via the unauthenticated /call endpoint using function token auth.
func (f *FunctionClient) Call(ctx context.Context, name string, version *int, event any, token, idempotencyKey string) (*FunctionCallResult, error) {
	if name == "" {
		return nil, fmt.Errorf("function name is required")
	}
	path := "/service/automate/functions/call/" + url.PathEscape(name)
	if version != nil {
		path += "/v/" + strconv.Itoa(*version)
	}
	req := f.c.RawRequest().SetContext(ctx).SetBody(event)
	if token != "" {
		req.SetHeader("X-Webhook-Token", token)
	}
	if idempotencyKey != "" {
		req.SetHeader("Idempotency-Key", idempotencyKey)
	}
	resp, err := req.Post(path)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	var result FunctionCallResult
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
