package client

import (
	"context"
	"fmt"
	"net/url"
)

// WorkflowClient provides access to the workflow management and execution API.
type WorkflowClient struct {
	c *Client
}

// WorkflowInfo is a workflow list entry.
type WorkflowInfo struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Describe       string `json:"describe"`
	Enabled        bool   `json:"enabled"`
	Resumable      bool   `json:"resumable"`
	MaxConcurrency int    `json:"max_concurrency"`
}

// WorkflowListResult contains listed workflows.
type WorkflowListResult struct {
	Workflows []WorkflowInfo `json:"workflows"`
}

// WorkflowApplyResult is returned after applying a YAML definition.
type WorkflowApplyResult struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// WorkflowExportResult holds exported YAML text.
type WorkflowExportResult struct {
	YAML string `json:"yaml"`
}

// WorkflowRunResult is returned when starting an asynchronous run.
type WorkflowRunResult struct {
	RunID int64 `json:"run_id"`
}

// WorkflowRunInfo is a single run history entry.
type WorkflowRunInfo struct {
	ID           int64  `json:"id"`
	WorkflowName string `json:"workflow_name"`
	Status       int    `json:"status"`
	TriggerType  string `json:"trigger_type"`
	Error        string `json:"error,omitempty"`
}

// WorkflowRunsResult contains run history for a workflow.
type WorkflowRunsResult struct {
	Runs []WorkflowRunInfo `json:"runs"`
}

// Apply upserts a workflow definition from YAML bytes.
func (w *WorkflowClient) Apply(ctx context.Context, yamlBytes []byte) (*WorkflowApplyResult, error) {
	var result WorkflowApplyResult
	err := w.c.Post(ctx, "/service/workflow/apply", map[string]string{"yaml": string(yamlBytes)}, &result)
	return &result, err
}

// List returns stored workflow definitions.
func (w *WorkflowClient) List(ctx context.Context) (*WorkflowListResult, error) {
	var result WorkflowListResult
	err := w.c.Get(ctx, "/service/workflow/list", &result)
	return &result, err
}

// Get returns a workflow definition by name.
func (w *WorkflowClient) Get(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	path := "/service/workflow/get/" + url.PathEscape(name)
	err := w.c.Get(ctx, path, &result)
	return result, err
}

// Export returns the YAML representation of a workflow.
func (w *WorkflowClient) Export(ctx context.Context, name string) (*WorkflowExportResult, error) {
	var result WorkflowExportResult
	path := "/service/workflow/export/" + url.PathEscape(name)
	err := w.c.Get(ctx, path, &result)
	return &result, err
}

// Delete removes a workflow definition by name.
func (w *WorkflowClient) Delete(ctx context.Context, name string) error {
	path := "/service/workflow/delete/" + url.PathEscape(name)
	var result map[string]any
	return w.c.Delete(ctx, path, nil, &result)
}

// Run starts an asynchronous workflow run and returns the run ID.
func (w *WorkflowClient) Run(ctx context.Context, name string, input map[string]any) (*WorkflowRunResult, error) {
	if name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}
	if input == nil {
		input = map[string]any{}
	}
	var result WorkflowRunResult
	err := w.c.Post(ctx, "/service/workflow/run", map[string]any{
		"name":  name,
		"input": input,
	}, &result)
	return &result, err
}

// Runs lists recent runs for a workflow.
func (w *WorkflowClient) Runs(ctx context.Context, name string) (*WorkflowRunsResult, error) {
	var result WorkflowRunsResult
	path := "/service/workflow/runs/" + url.PathEscape(name)
	err := w.c.Get(ctx, path, &result)
	return &result, err
}
