package dozzle

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "dozzle"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
	TokenKey    = "token"
)

// Dozzle is an HTTP client for Dozzle health/version endpoints.
type Dozzle struct {
	c *resty.Client
}

// GetClient builds a Dozzle client from vendors.dozzle config.
// Returns nil when endpoint is not configured.
func GetClient() *Dozzle {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	username, _ := providers.GetConfig(ID, UsernameKey)
	password, _ := providers.GetConfig(ID, PasswordKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewDozzle(endpoint.String(), username.String(), password.String(), token.String())
}

// NewDozzle creates a Dozzle client with optional Basic or Bearer auth.
// Returns nil when endpoint is empty.
func NewDozzle(endpoint, username, password, token string) *Dozzle {
	if endpoint == "" {
		return nil
	}
	v := &Dozzle{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	} else if username != "" || password != "" {
		v.c.SetBasicAuth(username, password)
	}
	return v
}

// Health reports whether Dozzle /healthcheck returns 2xx.
func (d *Dozzle) Health(ctx context.Context) error {
	if d == nil || d.c == nil {
		return fmt.Errorf("dozzle: not configured")
	}
	resp, err := d.c.R().SetContext(ctx).Get("/healthcheck")
	if err != nil {
		return fmt.Errorf("dozzle health: %w", err)
	}
	return checkStatus(resp, "health")
}

// Version returns the Dozzle version string from /api/version.
func (d *Dozzle) Version(ctx context.Context) (*VersionInfo, error) {
	resp, err := d.c.R().SetContext(ctx).Get("/api/version")
	if err != nil {
		return nil, fmt.Errorf("dozzle version: %w", err)
	}
	if err := checkStatus(resp, "version"); err != nil {
		return nil, err
	}
	return &VersionInfo{Version: strings.TrimSpace(string(resp.Bytes()))}, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("dozzle %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("dozzle %s: status %d", op, code)
}
