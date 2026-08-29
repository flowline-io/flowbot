package scanopy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "scanopy"
	EndpointKey = "endpoint"
	APIKeyKey   = "api_key"
)

// Scanopy is an HTTP client for the Scanopy REST API (/api/v1).
type Scanopy struct {
	c *resty.Client
}

// GetClient builds a Scanopy client from providers.scanopy config.
// Returns nil when endpoint is not configured.
func GetClient() *Scanopy {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, APIKeyKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewScanopy(endpoint.String(), apiKey.String())
}

// NewScanopy creates a Scanopy client with Bearer user API key auth.
// Returns nil when endpoint is empty.
func NewScanopy(endpoint, apiKey string) *Scanopy {
	if endpoint == "" {
		return nil
	}
	v := &Scanopy{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(strings.TrimRight(endpoint, "/"))
	if apiKey != "" {
		v.c.SetAuthToken(apiKey)
	}
	return v
}

// Health reports whether GET /api/version succeeds.
func (s *Scanopy) Health(ctx context.Context) error {
	if s == nil || s.c == nil {
		return fmt.Errorf("scanopy: not configured")
	}
	_, err := s.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("scanopy health: %w", err)
	}
	return nil
}

// GetVersion returns API and server versions from GET /api/version.
func (s *Scanopy) GetVersion(ctx context.Context) (*VersionInfo, error) {
	resp, err := s.c.R().SetContext(ctx).Get("/api/version")
	if err != nil {
		return nil, fmt.Errorf("scanopy version: %w", err)
	}
	if err := checkStatus(resp, "version"); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := sonic.Unmarshal(resp.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("scanopy version decode: %w", err)
	}
	if _, hasSuccess := raw["success"]; hasSuccess {
		var env Envelope[VersionInfo]
		if err := sonic.Unmarshal(resp.Bytes(), &env); err != nil {
			return nil, fmt.Errorf("scanopy version decode: %w", err)
		}
		if !env.Success {
			return nil, apiError("version", env.Error)
		}
		info := env.Data
		if info.ServerVersion == "" {
			info.ServerVersion = env.Meta.ServerVersion
		}
		if info.APIVersion == 0 {
			info.APIVersion = env.Meta.APIVersion
		}
		return &info, nil
	}
	var direct VersionInfo
	if err := sonic.Unmarshal(resp.Bytes(), &direct); err != nil {
		return nil, fmt.Errorf("scanopy version decode: %w", err)
	}
	return &direct, nil
}

// ListNetworks returns networks the API key can access.
func (s *Scanopy) ListNetworks(ctx context.Context, params ListParams) (*Page[Network], error) {
	return listPage[Network](ctx, s, "/api/v1/networks", params, "list networks")
}

// ListHosts returns hosts, optionally filtered by network, search, or staleness.
func (s *Scanopy) ListHosts(ctx context.Context, params ListParams) (*Page[Host], error) {
	return listPage[Host](ctx, s, "/api/v1/hosts", params, "list hosts")
}

// GetHost returns a single host by ID (includes children by default).
func (s *Scanopy) GetHost(ctx context.Context, id string) (*Host, error) {
	if id == "" {
		return nil, fmt.Errorf("scanopy: host id required")
	}
	path := "/api/v1/hosts/" + url.PathEscape(id)
	resp, err := s.c.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, fmt.Errorf("scanopy get host: %w", err)
	}
	if err := checkStatus(resp, "get host"); err != nil {
		return nil, err
	}
	var env Envelope[Host]
	if err := sonic.Unmarshal(resp.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("scanopy get host decode: %w", err)
	}
	if !env.Success {
		return nil, apiError("get host", env.Error)
	}
	return &env.Data, nil
}

// ListServices returns services, optionally filtered by network or host.
func (s *Scanopy) ListServices(ctx context.Context, params ListParams) (*Page[Service], error) {
	return listPage[Service](ctx, s, "/api/v1/services", params, "list services")
}

// ListDaemons returns scanning daemons.
func (s *Scanopy) ListDaemons(ctx context.Context, params ListParams) (*Page[Daemon], error) {
	return listPage[Daemon](ctx, s, "/api/v1/daemons", params, "list daemons")
}

func listPage[T any](ctx context.Context, s *Scanopy, path string, params ListParams, op string) (*Page[T], error) {
	req := s.c.R().SetContext(ctx)
	applyListQuery(req, params)
	resp, err := req.Get(path)
	if err != nil {
		return nil, fmt.Errorf("scanopy %s: %w", op, err)
	}
	if err := checkStatus(resp, op); err != nil {
		return nil, err
	}
	var env Envelope[[]T]
	if err := sonic.Unmarshal(resp.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("scanopy %s decode: %w", op, err)
	}
	if !env.Success {
		return nil, apiError(op, env.Error)
	}
	page := &Page[T]{Items: env.Data, Meta: env.Meta}
	if env.Meta.Pagination != nil {
		page.Pagination = *env.Meta.Pagination
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return page, nil
}

func applyListQuery(req *resty.Request, p ListParams) {
	if p.NetworkID != "" {
		req.SetQueryParam("network_id", p.NetworkID)
	}
	if p.HostID != "" {
		req.SetQueryParam("host_id", p.HostID)
	}
	if p.Search != "" {
		req.SetQueryParam("search", p.Search)
	}
	if p.Limit != nil {
		req.SetQueryParam("limit", strconv.Itoa(*p.Limit))
	}
	if p.Offset > 0 {
		req.SetQueryParam("offset", strconv.Itoa(p.Offset))
	}
}

func apiError(op, msg string) error {
	if msg == "" {
		msg = "request failed"
	}
	return fmt.Errorf("scanopy %s: %s", op, msg)
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("scanopy %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	var env Envelope[any]
	if err := sonic.Unmarshal(resp.Bytes(), &env); err == nil && env.Error != "" {
		return fmt.Errorf("scanopy %s: status %d: %s", op, code, env.Error)
	}
	return fmt.Errorf("scanopy %s: status %d", op, code)
}
