package wakapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "wakapi"
	EndpointKey = "endpoint"
	APIKeyKey   = "api_key"
)

// Wakapi is an HTTP client for the Wakapi API.
type Wakapi struct {
	c *resty.Client
}

// GetClient builds a Wakapi client from vendors.wakapi config.
// Returns nil when endpoint is not configured.
func GetClient() *Wakapi {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, APIKeyKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewWakapi(endpoint.String(), apiKey.String())
}

// NewWakapi creates a Wakapi client. Auth uses Basic base64(api_key).
// Returns nil when endpoint is empty.
func NewWakapi(endpoint, apiKey string) *Wakapi {
	if endpoint == "" {
		return nil
	}
	v := &Wakapi{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if apiKey != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(apiKey))
		v.c.SetHeader("Authorization", "Basic "+encoded)
	}
	return v
}

// GetSummary returns activity summary for an interval (e.g. "today").
func (w *Wakapi) GetSummary(ctx context.Context, interval string) (*Summary, error) {
	if interval == "" {
		interval = "today"
	}
	resp, err := w.c.R().
		SetContext(ctx).
		SetQueryParam("interval", interval).
		Get("/api/summary")
	if err != nil {
		return nil, fmt.Errorf("wakapi summary: %w", err)
	}
	if err := checkStatus(resp, "summary"); err != nil {
		return nil, err
	}
	var result Summary
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi summary decode: %w", err)
	}
	return &result, nil
}

// ListProjects returns tracked projects (WakaTime-compatible).
func (w *Wakapi) ListProjects(ctx context.Context) ([]Project, error) {
	resp, err := w.c.R().
		SetContext(ctx).
		Get("/api/compat/wakatime/v1/users/current/projects")
	if err != nil {
		return nil, fmt.Errorf("wakapi list projects: %w", err)
	}
	if err := checkStatus(resp, "list projects"); err != nil {
		return nil, err
	}
	var result ProjectsResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("wakapi list projects decode: %w", err)
	}
	return result.Data, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("wakapi %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("wakapi %s: status %d", op, code)
}
