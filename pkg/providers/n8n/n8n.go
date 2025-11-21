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
