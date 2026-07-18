package netalertx

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
	ID          = "netalertx"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

// NetAlertX is an HTTP client for the NetAlertX REST API.
type NetAlertX struct {
	c *resty.Client
}

// GetClient builds a NetAlertX client from vendors.netalertx config.
// Returns nil when endpoint is not configured.
func GetClient() *NetAlertX {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewNetAlertX(endpoint.String(), token.String())
}

// NewNetAlertX creates a NetAlertX client with Bearer token auth.
// Returns nil when endpoint is empty.
func NewNetAlertX(endpoint, token string) *NetAlertX {
	if endpoint == "" {
		return nil
	}
	v := &NetAlertX{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetAuthToken(token)
	}
	return v
}

// Health reports whether the devices totals endpoint is reachable.
func (n *NetAlertX) Health(ctx context.Context) error {
	if n == nil || n.c == nil {
		return fmt.Errorf("netalertx: not configured")
	}
	resp, err := n.c.R().SetContext(ctx).Get("/devices/totals")
	if err != nil {
		return fmt.Errorf("netalertx health: %w", err)
	}
	return checkStatus(resp, "health")
}

// ListDevices returns all devices.
func (n *NetAlertX) ListDevices(ctx context.Context) ([]Device, error) {
	resp, err := n.c.R().SetContext(ctx).Get("/devices")
	if err != nil {
		return nil, fmt.Errorf("netalertx list devices: %w", err)
	}
	if err := checkStatus(resp, "list devices"); err != nil {
		return nil, err
	}
	var result DevicesResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("netalertx list devices decode: %w", err)
	}
	return result.Devices, nil
}

// GetTotals returns device category counts.
func (n *NetAlertX) GetTotals(ctx context.Context) (*Totals, error) {
	resp, err := n.c.R().SetContext(ctx).Get("/devices/totals")
	if err != nil {
		return nil, fmt.Errorf("netalertx totals: %w", err)
	}
	if err := checkStatus(resp, "totals"); err != nil {
		return nil, err
	}
	var raw []int
	if err := sonic.Unmarshal(resp.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("netalertx totals decode: %w", err)
	}
	t := &Totals{}
	if len(raw) > 0 {
		t.All = raw[0]
	}
	if len(raw) > 1 {
		t.Connected = raw[1]
	}
	if len(raw) > 2 {
		t.Favorites = raw[2]
	}
	if len(raw) > 3 {
		t.New = raw[3]
	}
	if len(raw) > 4 {
		t.Down = raw[4]
	}
	if len(raw) > 5 {
		t.Archived = raw[5]
	}
	return t, nil
}

// SearchDevices searches devices by MAC, name, or IP.
func (n *NetAlertX) SearchDevices(ctx context.Context, query string) ([]Device, error) {
	if query == "" {
		return nil, fmt.Errorf("netalertx: query is required")
	}
	resp, err := n.c.R().
		SetContext(ctx).
		SetBody(map[string]string{"query": query}).
		Post("/devices/search")
	if err != nil {
		return nil, fmt.Errorf("netalertx search devices: %w", err)
	}
	if err := checkStatus(resp, "search devices"); err != nil {
		return nil, err
	}
	var result SearchResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("netalertx search devices decode: %w", err)
	}
	return result.Devices, nil
}

// GetTopology returns network topology nodes and links.
func (n *NetAlertX) GetTopology(ctx context.Context) (*Topology, error) {
	resp, err := n.c.R().SetContext(ctx).Get("/devices/network/topology")
	if err != nil {
		return nil, fmt.Errorf("netalertx topology: %w", err)
	}
	if err := checkStatus(resp, "topology"); err != nil {
		return nil, err
	}
	var result Topology
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("netalertx topology decode: %w", err)
	}
	return &result, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("netalertx %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("netalertx %s: status %d", op, code)
}
