package traefik

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
	ID          = "traefik"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
)

// Traefik is an HTTP client for the Traefik API.
type Traefik struct {
	c *resty.Client
}

// GetClient builds a Traefik client from vendors.traefik config.
// Returns nil when endpoint is not configured.
func GetClient() *Traefik {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	username, _ := providers.GetConfig(ID, UsernameKey)
	password, _ := providers.GetConfig(ID, PasswordKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewTraefik(endpoint.String(), username.String(), password.String())
}

// NewTraefik creates a Traefik client with optional Basic auth.
// Returns nil when endpoint is empty.
func NewTraefik(endpoint, username, password string) *Traefik {
	if endpoint == "" {
		return nil
	}
	v := &Traefik{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if username != "" || password != "" {
		v.c.SetBasicAuth(username, password)
	}
	return v
}

// Overview returns Traefik overview statistics.
func (t *Traefik) Overview(ctx context.Context) (*Overview, error) {
	resp, err := t.c.R().SetContext(ctx).Get("/api/overview")
	if err != nil {
		return nil, fmt.Errorf("traefik overview: %w", err)
	}
	if err := checkStatus(resp, "overview"); err != nil {
		return nil, err
	}
	var result Overview
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("traefik overview decode: %w", err)
	}
	return &result, nil
}

// ListRouters returns HTTP routers.
func (t *Traefik) ListRouters(ctx context.Context) ([]Router, error) {
	resp, err := t.c.R().SetContext(ctx).Get("/api/http/routers")
	if err != nil {
		return nil, fmt.Errorf("traefik list routers: %w", err)
	}
	if err := checkStatus(resp, "list routers"); err != nil {
		return nil, err
	}
	var result []Router
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("traefik list routers decode: %w", err)
	}
	return result, nil
}

// ListServices returns HTTP services.
func (t *Traefik) ListServices(ctx context.Context) ([]Service, error) {
	resp, err := t.c.R().SetContext(ctx).Get("/api/http/services")
	if err != nil {
		return nil, fmt.Errorf("traefik list services: %w", err)
	}
	if err := checkStatus(resp, "list services"); err != nil {
		return nil, err
	}
	var result []Service
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("traefik list services decode: %w", err)
	}
	return result, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("traefik %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("traefik %s: status %d", op, code)
}
