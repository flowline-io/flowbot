package client

import (
	"context"
	"fmt"
	"net/url"
)

// PipelineClient provides access to the pipeline management and execution API.
type PipelineClient struct {
	c *Client
}

// PipelineInfo is a pipeline list entry.
type PipelineInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Resumable   bool     `json:"resumable"`
	Triggers    []string `json:"triggers"`
}

// PipelineListResult contains the list of pipelines.
type PipelineListResult struct {
	Pipelines []PipelineInfo `json:"pipelines"`
}

// PipelineApplyResult is returned after applying a YAML definition.
type PipelineApplyResult struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Version int    `json:"version"`
}

// PipelineExportResult holds exported YAML text.
type PipelineExportResult struct {
	YAML string `json:"yaml"`
}

// PipelineRunResult is returned when starting an asynchronous run.
type PipelineRunResult struct {
	RunID int64 `json:"run_id"`
}

// PipelineRunInfo is a single run history entry.
type PipelineRunInfo struct {
	ID            int64  `json:"id"`
	PipelineName  string `json:"pipeline_name"`
	Status        int    `json:"status"`
	TriggerSource string `json:"trigger_source"`
	EventID       string `json:"event_id,omitempty"`
	EventType     string `json:"event_type,omitempty"`
	Error         string `json:"error,omitempty"`
}

// PipelineRunsResult contains run history for a pipeline.
type PipelineRunsResult struct {
	Runs []PipelineRunInfo `json:"runs"`
}

// Apply upserts a pipeline definition from YAML bytes (draft + publish).
func (p *PipelineClient) Apply(ctx context.Context, yamlBytes []byte) (*PipelineApplyResult, error) {
	var result PipelineApplyResult
	err := p.c.Post(ctx, "/service/pipeline/apply", map[string]string{"yaml": string(yamlBytes)}, &result)
	return &result, err
}

// List returns published pipeline definitions.
func (p *PipelineClient) List(ctx context.Context) (*PipelineListResult, error) {
	var result PipelineListResult
	err := p.c.Get(ctx, "/service/pipeline/list", &result)
	return &result, err
}

// Get returns a pipeline definition by name.
func (p *PipelineClient) Get(ctx context.Context, name string) (map[string]any, error) {
	var result map[string]any
	path := "/service/pipeline/get/" + url.PathEscape(name)
	err := p.c.Get(ctx, path, &result)
	return result, err
}

// Export returns the published YAML representation of a pipeline.
func (p *PipelineClient) Export(ctx context.Context, name string) (*PipelineExportResult, error) {
	var result PipelineExportResult
	path := "/service/pipeline/export/" + url.PathEscape(name)
	err := p.c.Get(ctx, path, &result)
	return &result, err
}

// Delete removes a pipeline definition by name.
func (p *PipelineClient) Delete(ctx context.Context, name string) error {
	path := "/service/pipeline/delete/" + url.PathEscape(name)
	var result map[string]any
	return p.c.Delete(ctx, path, nil, &result)
}

// Run starts an asynchronous pipeline run with an optional event payload.
func (p *PipelineClient) Run(ctx context.Context, name string, event map[string]any) (*PipelineRunResult, error) {
	if name == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}
	if event == nil {
		event = map[string]any{}
	}
	var result PipelineRunResult
	err := p.c.Post(ctx, "/service/pipeline/run", map[string]any{"name": name, "event": event}, &result)
	return &result, err
}

// Runs returns run history for a pipeline.
func (p *PipelineClient) Runs(ctx context.Context, name string) (*PipelineRunsResult, error) {
	var result PipelineRunsResult
	path := "/service/pipeline/runs/" + url.PathEscape(name)
	err := p.c.Get(ctx, path, &result)
	return &result, err
}
