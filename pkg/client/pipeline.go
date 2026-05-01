package client

import "context"

// PipelineClient provides access to the pipeline API.
type PipelineClient struct {
	c *Client
}

// PipelineInfo is a pipeline metadata record.
type PipelineInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Trigger     struct {
		Event string `json:"event"`
	} `json:"trigger"`
	Steps []struct {
		Name       string         `json:"name"`
		Capability string         `json:"capability"`
		Operation  string         `json:"operation"`
		Params     map[string]any `json:"params"`
	} `json:"steps"`
}

// PipelineListResult contains the list of pipelines.
type PipelineListResult struct {
	Pipelines []PipelineInfo `json:"pipelines"`
}

// PipelineRunResult is the response from triggering a pipeline run.
type PipelineRunResult struct {
	Message string `json:"message"`
}

// List returns configured pipelines from the hub.
func (p *PipelineClient) List(ctx context.Context) (*PipelineListResult, error) {
	var result PipelineListResult
	err := p.c.Get(ctx, "/service/pipeline/list", &result)
	return &result, err
}

// Run triggers a pipeline run by name.
func (p *PipelineClient) Run(ctx context.Context, name string) (*PipelineRunResult, error) {
	var result PipelineRunResult
	err := p.c.Post(ctx, "/service/pipeline/run", map[string]string{"name": name}, &result)
	return &result, err
}
