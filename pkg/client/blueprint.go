package client

import (
	"context"
	"errors"
	"net/url"
)

// BlueprintClient accesses the pipeline blueprint catalog API.
type BlueprintClient struct {
	c *Client
}

// BlueprintInfo is a catalog list entry.
type BlueprintInfo struct {
	Source        string   `json:"source"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Missing       []string `json:"missing"`
	InstanceCount int      `json:"instance_count"`
	Hash          string   `json:"hash"`
}

// BlueprintListResult is the catalog list payload.
type BlueprintListResult struct {
	Blueprints []BlueprintInfo `json:"blueprints"`
}

// BlueprintImportResult is returned after importing YAML.
type BlueprintImportResult struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source"`
	Hash   string `json:"hash"`
}

// BlueprintInstantiateResult is returned after creating a linked pipeline.
type BlueprintInstantiateResult struct {
	Name    string `json:"name"`
	ID      int64  `json:"id"`
	Version int    `json:"version"`
}

func blueprintPath(source, id, suffix string) string {
	p := "/service/automate/pipeline/blueprints/" + url.PathEscape(source) + "/" + url.PathEscape(id)
	if suffix != "" {
		p += "/" + suffix
	}
	return p
}

// List returns official and imported blueprints.
func (b *BlueprintClient) List(ctx context.Context) (*BlueprintListResult, error) {
	var result BlueprintListResult
	err := b.c.Get(ctx, "/service/automate/pipeline/blueprints", &result)
	return &result, err
}

// Import adds a user template from YAML.
func (b *BlueprintClient) Import(ctx context.Context, yamlBytes []byte) (*BlueprintImportResult, error) {
	var result BlueprintImportResult
	err := b.c.Post(ctx, "/service/automate/pipeline/blueprints/import", map[string]string{"yaml": string(yamlBytes)}, &result)
	return &result, err
}

// Export returns the template YAML.
func (b *BlueprintClient) Export(ctx context.Context, source, id string) (*PipelineExportResult, error) {
	var result PipelineExportResult
	err := b.c.Get(ctx, blueprintPath(source, id, "export"), &result)
	return &result, err
}

// Replace updates an imported template.
func (b *BlueprintClient) Replace(ctx context.Context, id string, yamlBytes []byte) (*BlueprintImportResult, error) {
	var result BlueprintImportResult
	err := b.c.Put(ctx, blueprintPath("upload", id, ""), map[string]string{"yaml": string(yamlBytes)}, &result)
	return &result, err
}

// Delete removes an imported template.
func (b *BlueprintClient) Delete(ctx context.Context, id string) error {
	var result map[string]any
	return b.c.Delete(ctx, blueprintPath("upload", id, ""), nil, &result)
}

// Instantiate creates a linked pipeline from a template.
func (b *BlueprintClient) Instantiate(ctx context.Context, source, id, name string, inputs map[string]any, enable bool) (*BlueprintInstantiateResult, error) {
	if name == "" {
		return nil, errors.New("pipeline name is required")
	}
	if inputs == nil {
		inputs = map[string]any{}
	}
	var result BlueprintInstantiateResult
	err := b.c.Post(ctx, blueprintPath(source, id, "instantiate"), map[string]any{
		"name":   name,
		"inputs": inputs,
		"enable": enable,
	}, &result)
	return &result, err
}

// UpdateInputs rematerializes a linked pipeline.
func (b *BlueprintClient) UpdateInputs(ctx context.Context, pipelineName string, inputs map[string]any) (*BlueprintInstantiateResult, error) {
	var result BlueprintInstantiateResult
	path := "/service/automate/pipeline/instances/" + url.PathEscape(pipelineName) + "/inputs"
	err := b.c.Post(ctx, path, map[string]any{"inputs": inputs}, &result)
	return &result, err
}

// ConfirmUpdate rematerializes from the current catalog template.
func (b *BlueprintClient) ConfirmUpdate(ctx context.Context, pipelineName string) (*BlueprintInstantiateResult, error) {
	var result BlueprintInstantiateResult
	path := "/service/automate/pipeline/instances/" + url.PathEscape(pipelineName) + "/update"
	err := b.c.Post(ctx, path, map[string]any{}, &result)
	return &result, err
}

// TakeControl detaches a linked pipeline from its blueprint.
func (b *BlueprintClient) TakeControl(ctx context.Context, pipelineName string) error {
	var result map[string]any
	path := "/service/automate/pipeline/instances/" + url.PathEscape(pipelineName) + "/take-control"
	return b.c.Post(ctx, path, map[string]any{}, &result)
}
