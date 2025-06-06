package uptimekuma

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"resty.dev/v3"
)

const (
	ID          = "uptimekuma"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type UptimeKuma struct {
	c *resty.Client
}

func GetClient() *UptimeKuma {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)

	return NewUptimeKuma(endpoint.String(), tokenKey.String())
}

func NewUptimeKuma(endpoint string, token string) *UptimeKuma {
	v := &UptimeKuma{}

	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetDisableWarn(true)
	v.c.SetBasicAuth("", token)

	return v
}

func (i *UptimeKuma) Metrics() (map[string]*dto.MetricFamily, error) {
	resp, err := i.c.R().
		Get("/metrics")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	return metricFamilies, nil
}
