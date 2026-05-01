package client

import "context"

// WorkflowClient provides access to the workflow execution API.
type WorkflowClient struct {
	c *Client
}

// WorkflowRunResult is the response from a workflow run.
type WorkflowRunResult struct {
	Message string `json:"message"`
}

// RunFile uploads a workflow YAML file and runs it on the server.
func (w *WorkflowClient) RunFile(ctx context.Context, filePath string) (*WorkflowRunResult, error) {
	var result WorkflowRunResult
	err := w.c.Post(ctx, "/service/workflow/run", map[string]string{"file": filePath}, &result)
	return &result, err
}
