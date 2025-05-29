package victoriametrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID = "victoriametrics"
)

type VictoriaMetrics struct {
	c *resty.Client
}

func NewVictoriaMetrics() *VictoriaMetrics {
	v := &VictoriaMetrics{}
	v.c = resty.New()
	v.c.SetTimeout(time.Minute)
	v.c.SetBaseURL(config.App.Metrics.Endpoint)

	return v
}

func (i *VictoriaMetrics) Query(expression string) (*MetricsResponse, error) {
	resp, err := i.c.R().
		SetResult(&MetricsResponse{}).
		SetQueryParam("query", expression).
		Get("/prometheus/api/v1/query")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*MetricsResponse)
		return result, nil
	} else {
		return nil, fmt.Errorf("%d, %s", resp.StatusCode(), utils.BytesToString(resp.Bytes()))
	}
}
