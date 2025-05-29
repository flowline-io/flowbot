package wallos

import (
	"context"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	"resty.dev/v3"
)

const (
	ID          = "wallos"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Wallos struct {
	ctx    context.Context
	c      *resty.Client
	apiKey string
}

func GetClient() *Wallos {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, ApikeyKey)

	return NewWallos(endpoint.String(), apiKey.String())
}

func NewWallos(endpoint string, apiKey string) *Wallos {
	v := &Wallos{apiKey: apiKey}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}

// GetSubscriptions Get wallos subscriptions
// Document: https://github.com/ellite/Wallos/blob/main/api/subscriptions/get_subscriptions.php
func (i *Wallos) GetSubscriptions(ctx context.Context) (*GetSubscriptionsResponse, error) {
	resp, err := i.c.R().
		SetContext(ctx).
		SetQueryParam("api_key", i.apiKey).
		SetResult(&GetSubscriptionsResponse{}).
		Get("/api/subscriptions/get_subscriptions.php")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*GetSubscriptionsResponse)
		return result, nil
	}
	return nil, nil
}

// GetMonthlyCost Get monthly cost
// Document: https://github.com/ellite/Wallos/blob/main/api/subscriptions/get_monthly_cost.php
func (i *Wallos) GetMonthlyCost(ctx context.Context, year, month int32) (*GetMonthlyCostResponse, error) {
	resp, err := i.c.R().
		SetContext(ctx).
		SetQueryParam("api_key", i.apiKey).
		SetQueryParam("year", string(year)).
		SetQueryParam("month", string(month)).
		SetResult(&GetMonthlyCostResponse{}).
		Get("/api/subscriptions/get_monthly_cost.php")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*GetMonthlyCostResponse)
		return result, nil
	}
	return nil, nil
}
