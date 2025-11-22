package n8n

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID          = "n8n"
	EndpointKey = "endpoint"
	ApiKeyKey   = "api_key"
)

type N8N struct {
	c *resty.Client
}

func GetClient() (*N8N, error) {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, ApiKeyKey)
	if endpoint.String() == "" {
		return nil, fmt.Errorf("n8n disabled")
	}

	return NewN8N(endpoint.String(), apiKey.String()), nil
}

func NewN8N(endpoint string, apiKey string) *N8N {
	v := &N8N{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if apiKey != "" {
		v.c.SetHeader("X-N8N-API-KEY", apiKey)
	}

	return v
}

// ListWorkflows retrieves all workflows
func (v *N8N) ListWorkflows() ([]*Workflow, error) {
	var workflows []*Workflow
	resp, err := v.c.R().
		SetResult(&workflows).
		Get("/workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	if resp.StatusCode() == http.StatusOK {
		return workflows, nil
	} else {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// GetWorkflow retrieves a workflow by ID
func (v *N8N) GetWorkflow(id string) (*Workflow, error) {
	resp, err := v.c.R().
		SetResult(&Workflow{}).
		SetPathParam("id", id).
		Get("/workflows/{id}")
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Workflow), nil
	} else {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// CreateWorkflow creates a new workflow
func (v *N8N) CreateWorkflow(workflow *Workflow) (*Workflow, error) {
	resp, err := v.c.R().
		SetResult(&Workflow{}).
		SetBody(workflow).
		Post("/workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusCreated || resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Workflow), nil
	} else {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// UpdateWorkflow updates an existing workflow
func (v *N8N) UpdateWorkflow(id string, workflow *Workflow) (*Workflow, error) {
	resp, err := v.c.R().
		SetResult(&Workflow{}).
		SetPathParam("id", id).
		SetBody(workflow).
		Put("/workflows/{id}")
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Workflow), nil
	} else {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// DeleteWorkflow deletes a workflow by ID
func (v *N8N) DeleteWorkflow(id string) error {
	resp, err := v.c.R().
		SetPathParam("id", id).
		Delete("/workflows/{id}")
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusNoContent {
		return nil
	} else {
		return fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// ActivateWorkflow activates a workflow
func (v *N8N) ActivateWorkflow(id string) error {
	resp, err := v.c.R().
		SetPathParam("id", id).
		Post("/workflows/{id}/activate")
	if err != nil {
		return fmt.Errorf("failed to activate workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusNoContent {
		return nil
	} else {
		return fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// DeactivateWorkflow deactivates a workflow
func (v *N8N) DeactivateWorkflow(id string) error {
	resp, err := v.c.R().
		SetPathParam("id", id).
		Post("/workflows/{id}/deactivate")
	if err != nil {
		return fmt.Errorf("failed to deactivate workflow: %w", err)
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusNoContent {
		return nil
	} else {
		return fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}

// ExecuteWorkflow executes a workflow via webhook with optional input data
func (v *N8N) ExecuteWorkflow(id string, data map[string]interface{}) error {
	// Get workflow to find webhook node
	workflow, err := v.GetWorkflow(id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Find webhook node (usually the first trigger node)
	var webhookPath string
	var webhookID string
	for _, node := range workflow.Nodes {
		// Check if node is a webhook trigger
		if node.Type == "n8n-nodes-base.webhook" || node.Type == "n8n-nodes-base.webhookV2" {
			// Extract webhook path from parameters
			if node.Parameters != nil {
				if path, ok := node.Parameters["path"].(string); ok && path != "" {
					webhookPath = path
				}
			}
			if node.WebhookID != "" {
				webhookID = node.WebhookID
			}
			break
		}
	}

	if webhookPath == "" && webhookID == "" {
		return fmt.Errorf("workflow does not have a webhook trigger node or webhook path is not configured")
	}

	// Build webhook URL
	// n8n webhook URL format: /webhook/<path> or /webhook/<id>
	var webhookURL string
	if webhookPath != "" {
		// Remove leading slash if present
		if len(webhookPath) > 0 && webhookPath[0] == '/' {
			webhookPath = webhookPath[1:]
		}
		webhookURL = fmt.Sprintf("/webhook/%s", webhookPath)
	} else {
		webhookURL = fmt.Sprintf("/webhook/%s", webhookID)
	}

	// Execute workflow via webhook
	req := v.c.R()
	if data != nil {
		req = req.SetBody(data)
	}

	resp, err := req.Post(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to execute workflow via webhook: %w", err)
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusAccepted {
		return nil
	} else {
		return fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), resp.String())
	}
}
