package uptimekuma

import (
	"context"
	"fmt"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "uptimekuma"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

// UptimeKuma is an HTTP client for Uptime Kuma Prometheus metrics.
type UptimeKuma struct {
	c *resty.Client
}

// GetClient builds an Uptime Kuma client from vendors.uptimekuma config.
// Returns nil when endpoint is not configured.
func GetClient() *UptimeKuma {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewUptimeKuma(endpoint.String(), tokenKey.String())
}

// NewUptimeKuma creates an Uptime Kuma client with Basic auth (empty user + token).
// Returns nil when endpoint is empty.
func NewUptimeKuma(endpoint, token string) *UptimeKuma {
	if endpoint == "" {
		return nil
	}
	v := &UptimeKuma{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetBasicAuth("", token)
	return v
}

// Health reports whether the /metrics endpoint is reachable with a 2xx status.
func (i *UptimeKuma) Health(ctx context.Context) error {
	resp, err := i.c.R().SetContext(ctx).Get("/metrics")
	if err != nil {
		return fmt.Errorf("uptimekuma health: %w", err)
	}
	code := resp.StatusCode()
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return fmt.Errorf("uptimekuma health: status %d", code)
	}
	return nil
}

// Metrics fetches and parses Prometheus metrics from /metrics.
func (i *UptimeKuma) Metrics(ctx context.Context) (map[string]*dto.MetricFamily, error) {
	resp, err := i.c.R().SetContext(ctx).Get("/metrics")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	code := resp.StatusCode()
	if code < http.StatusOK || code >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("uptimekuma metrics: status %d", code)
	}

	parser := expfmt.NewTextParser(model.LegacyValidation)
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}
	return metricFamilies, nil
}
