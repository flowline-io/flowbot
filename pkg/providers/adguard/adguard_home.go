package adguard

import (
	"context"
	"fmt"
	openapi "github.com/flowline-io/sdk-adguard-home-api"
)

const (
	ID          = "adguard_home"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
)

type AdGuardHome struct {
	ctx context.Context
	c   *openapi.APIClient
}

func NewAdGuardHome(endpoint string, username string, password string) *AdGuardHome {
	v := &AdGuardHome{}

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{{URL: endpoint}}
	v.c = openapi.NewAPIClient(cfg)

	ctx := context.WithValue(context.Background(), openapi.ContextServerIndex, 0)
	ctx = context.WithValue(ctx, openapi.ContextBasicAuth, openapi.BasicAuth{UserName: username, Password: password})
	v.ctx = ctx

	return v
}

func (v *AdGuardHome) GetStatus() (*openapi.ServerStatus, error) {
	stats, _, err := v.c.GlobalAPI.Status(v.ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get status, %w", err)
	}
	return stats, nil
}

func (v *AdGuardHome) GetStats() (*openapi.Stats, error) {
	stats, _, err := v.c.StatsAPI.Stats(v.ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats, %w", err)
	}

	return stats, nil
}
