package wallos

import (
	"context"
	"fmt"
	openapi "github.com/flowline-io/sdk-wallos-api"
)

const (
	ID          = "wallos"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Wallos struct {
	ctx    context.Context
	c      *openapi.APIClient
	apiKey string
}

func NewWallos(endpoint string, apiKey string) *Wallos {
	v := &Wallos{}

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{{URL: endpoint}}
	v.c = openapi.NewAPIClient(cfg)

	ctx := context.WithValue(context.Background(), openapi.ContextServerIndex, 0)
	v.ctx = ctx
	v.apiKey = apiKey

	return v
}

func (i *Wallos) GetSubscriptions() ([]openapi.ApiSubscriptionsGetSubscriptionsPhpGet200ResponseSubscriptionsInner, error) {
	resp, _, err := i.c.DefaultAPI.ApiSubscriptionsGetSubscriptionsPhpGet(i.ctx).ApiKey(i.apiKey).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	if resp.GetSuccess() != true {
		return nil, fmt.Errorf("failed to get subscriptions: %s", resp.GetTitle())
	}

	return resp.GetSubscriptions(), nil
}
