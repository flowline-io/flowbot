package grafana

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "grafana"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

// Grafana is an HTTP client for the Grafana API.
type Grafana struct {
	c *resty.Client
}

// GetClient builds a Grafana client from vendors.grafana config.
// Returns nil when endpoint is not configured.
func GetClient() *Grafana {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewGrafana(endpoint.String(), token.String())
}

// NewGrafana creates a Grafana client with a Bearer API token.
// Returns nil when endpoint is empty.
func NewGrafana(endpoint, token string) *Grafana {
	if endpoint == "" {
		return nil
	}
	v := &Grafana{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	}
	return v
}

// Health returns Grafana instance health.
func (g *Grafana) Health(ctx context.Context) (*Health, error) {
	resp, err := g.c.R().SetContext(ctx).Get("/api/health")
	if err != nil {
		return nil, fmt.Errorf("grafana health: %w", err)
	}
	if err := checkStatus(resp, "health"); err != nil {
		return nil, err
	}
	var result Health
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("grafana health decode: %w", err)
	}
	return &result, nil
}

// ListDatasources returns configured data sources.
func (g *Grafana) ListDatasources(ctx context.Context) ([]Datasource, error) {
	resp, err := g.c.R().SetContext(ctx).Get("/api/datasources")
	if err != nil {
		return nil, fmt.Errorf("grafana list datasources: %w", err)
	}
	if err := checkStatus(resp, "list datasources"); err != nil {
		return nil, err
	}
	var result []Datasource
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("grafana list datasources decode: %w", err)
	}
	return result, nil
}

// SearchDashboards searches dashboards by optional query.
func (g *Grafana) SearchDashboards(ctx context.Context, query string) ([]DashboardHit, error) {
	req := g.c.R().SetContext(ctx).SetQueryParam("type", "dash-db")
	if query != "" {
		req.SetQueryParam("query", query)
	}
	resp, err := req.Get("/api/search")
	if err != nil {
		return nil, fmt.Errorf("grafana search dashboards: %w", err)
	}
	if err := checkStatus(resp, "search dashboards"); err != nil {
		return nil, err
	}
	var result []DashboardHit
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("grafana search dashboards decode: %w", err)
	}
	return result, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("grafana %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("grafana %s: status %d", op, code)
}
