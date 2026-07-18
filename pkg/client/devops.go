package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// DevopsClient provides access to the devops aggregator capability API.
type DevopsClient struct {
	c *Client
}

// DevopsStatusResult holds status extracted from InvokeResult.
type DevopsStatusResult struct {
	Status capability.DevopsStatus `json:"data"`
}

// DevopsSystemsResult holds Beszel systems from InvokeResult.
type DevopsSystemsResult struct {
	Items []*capability.DevopsSystem `json:"data"`
	Page  *capability.PageInfo       `json:"page"`
}

// DevopsSystemResult holds a single Beszel system from InvokeResult.
type DevopsSystemResult struct {
	Item capability.DevopsSystem `json:"data"`
}

// DevopsHealthyResult holds a simple healthy flag from InvokeResult.
type DevopsHealthyResult struct {
	Data map[string]bool `json:"data"`
}

// DevopsMetricsResult holds Uptime Kuma metric family summaries.
type DevopsMetricsResult struct {
	Items []*capability.DevopsMetricFamily `json:"data"`
	Page  *capability.PageInfo             `json:"page"`
}

// DevopsTraefikOverviewResult holds Traefik overview from InvokeResult.
type DevopsTraefikOverviewResult struct {
	Overview capability.DevopsTraefikOverview `json:"data"`
}

// DevopsRoutersResult holds Traefik routers from InvokeResult.
type DevopsRoutersResult struct {
	Items []*capability.DevopsRouter `json:"data"`
	Page  *capability.PageInfo       `json:"page"`
}

// DevopsServicesResult holds Traefik services from InvokeResult.
type DevopsServicesResult struct {
	Items []*capability.DevopsService `json:"data"`
	Page  *capability.PageInfo        `json:"page"`
}

// DevopsGrafanaHealthResult holds Grafana health from InvokeResult.
type DevopsGrafanaHealthResult struct {
	Health capability.DevopsGrafanaHealth `json:"data"`
}

// DevopsDatasourcesResult holds Grafana datasources from InvokeResult.
type DevopsDatasourcesResult struct {
	Items []*capability.DevopsDatasource `json:"data"`
	Page  *capability.PageInfo           `json:"page"`
}

// DevopsDashboardsResult holds Grafana dashboards from InvokeResult.
type DevopsDashboardsResult struct {
	Items []*capability.DevopsDashboard `json:"data"`
	Page  *capability.PageInfo          `json:"page"`
}

// DevopsWakapiSummaryResult holds Wakapi summary from InvokeResult.
type DevopsWakapiSummaryResult struct {
	Summary capability.DevopsWakapiSummary `json:"data"`
}

// DevopsWakapiProjectsResult holds Wakapi projects from InvokeResult.
type DevopsWakapiProjectsResult struct {
	Items []*capability.DevopsWakapiProject `json:"data"`
	Page  *capability.PageInfo              `json:"page"`
}

// DevopsDozzleInfoResult holds Dozzle health info from InvokeResult.
type DevopsDozzleInfoResult struct {
	Info capability.DevopsDozzleInfo `json:"data"`
}

// DevopsGrafanaQueryResult holds a Grafana query result from InvokeResult.
type DevopsGrafanaQueryResult struct {
	Result capability.DevopsGrafanaQueryResult `json:"data"`
}

// DevopsGrafanaQueryRequest is the request body for Grafana datasource queries.
type DevopsGrafanaQueryRequest struct {
	Backend       string `json:"backend"`
	Expr          string `json:"expr"`
	DatasourceUID string `json:"datasource_uid,omitzero"`
	From          string `json:"from,omitzero"`
	To            string `json:"to,omitzero"`
	MaxLines      int    `json:"max_lines,omitzero"`
}

// DevopsNetalertxDevicesResult holds NetAlertX devices from InvokeResult.
type DevopsNetalertxDevicesResult struct {
	Items []*capability.DevopsNetalertxDevice `json:"data"`
	Page  *capability.PageInfo                `json:"page"`
}

// DevopsNetalertxTotalsResult holds NetAlertX totals from InvokeResult.
type DevopsNetalertxTotalsResult struct {
	Totals capability.DevopsNetalertxTotals `json:"data"`
}

// DevopsNetalertxSearchRequest is the request body for NetAlertX device search.
type DevopsNetalertxSearchRequest struct {
	Query string `json:"query"`
}

// Status returns which devops backends are configured.
func (d *DevopsClient) Status(ctx context.Context) (*capability.DevopsStatus, error) {
	var result DevopsStatusResult
	if err := d.c.Get(ctx, "/service/devops/status", &result); err != nil {
		return nil, err
	}
	return &result.Status, nil
}

// BeszelListSystems lists Beszel systems.
func (d *DevopsClient) BeszelListSystems(ctx context.Context) (*DevopsSystemsResult, error) {
	var result DevopsSystemsResult
	if err := d.c.Get(ctx, "/service/devops/beszel/systems", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BeszelGetSystem returns one Beszel system.
func (d *DevopsClient) BeszelGetSystem(ctx context.Context, id string) (*capability.DevopsSystem, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	var result DevopsSystemResult
	path := "/service/devops/beszel/systems/" + url.PathEscape(id)
	if err := d.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// UptimekumaHealth checks Uptime Kuma reachability.
func (d *DevopsClient) UptimekumaHealth(ctx context.Context) (bool, error) {
	var result DevopsHealthyResult
	if err := d.c.Get(ctx, "/service/devops/uptimekuma/health", &result); err != nil {
		return false, err
	}
	return result.Data["healthy"], nil
}

// UptimekumaMetrics returns Uptime Kuma metric family summaries.
func (d *DevopsClient) UptimekumaMetrics(ctx context.Context) (*DevopsMetricsResult, error) {
	var result DevopsMetricsResult
	if err := d.c.Get(ctx, "/service/devops/uptimekuma/metrics", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TraefikOverview returns Traefik overview counts.
func (d *DevopsClient) TraefikOverview(ctx context.Context) (*capability.DevopsTraefikOverview, error) {
	var result DevopsTraefikOverviewResult
	if err := d.c.Get(ctx, "/service/devops/traefik/overview", &result); err != nil {
		return nil, err
	}
	return &result.Overview, nil
}

// TraefikListRouters lists Traefik HTTP routers.
func (d *DevopsClient) TraefikListRouters(ctx context.Context) (*DevopsRoutersResult, error) {
	var result DevopsRoutersResult
	if err := d.c.Get(ctx, "/service/devops/traefik/routers", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TraefikListServices lists Traefik HTTP services.
func (d *DevopsClient) TraefikListServices(ctx context.Context) (*DevopsServicesResult, error) {
	var result DevopsServicesResult
	if err := d.c.Get(ctx, "/service/devops/traefik/services", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GrafanaHealth returns Grafana health.
func (d *DevopsClient) GrafanaHealth(ctx context.Context) (*capability.DevopsGrafanaHealth, error) {
	var result DevopsGrafanaHealthResult
	if err := d.c.Get(ctx, "/service/devops/grafana/health", &result); err != nil {
		return nil, err
	}
	return &result.Health, nil
}

// GrafanaListDatasources lists Grafana datasources.
func (d *DevopsClient) GrafanaListDatasources(ctx context.Context) (*DevopsDatasourcesResult, error) {
	var result DevopsDatasourcesResult
	if err := d.c.Get(ctx, "/service/devops/grafana/datasources", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GrafanaSearchDashboards searches Grafana dashboards.
func (d *DevopsClient) GrafanaSearchDashboards(ctx context.Context, query string) (*DevopsDashboardsResult, error) {
	path := "/service/devops/grafana/dashboards"
	if query != "" {
		path += "?query=" + url.QueryEscape(query)
	}
	var result DevopsDashboardsResult
	if err := d.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GrafanaQuery runs a query against prometheus/alloy/loki/tempo/pyroscope via Grafana.
func (d *DevopsClient) GrafanaQuery(ctx context.Context, req DevopsGrafanaQueryRequest) (*capability.DevopsGrafanaQueryResult, error) {
	if req.Backend == "" {
		return nil, fmt.Errorf("backend is required")
	}
	if req.Expr == "" {
		return nil, fmt.Errorf("expr is required")
	}
	var result DevopsGrafanaQueryResult
	if err := d.c.Post(ctx, "/service/devops/grafana/query", &req, &result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// WakapiSummary returns a coding-stats summary.
func (d *DevopsClient) WakapiSummary(ctx context.Context, interval string) (*capability.DevopsWakapiSummary, error) {
	path := "/service/devops/wakapi/summary"
	if interval != "" {
		path += "?interval=" + url.QueryEscape(interval)
	}
	var result DevopsWakapiSummaryResult
	if err := d.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Summary, nil
}

// WakapiListProjects lists Wakapi projects.
func (d *DevopsClient) WakapiListProjects(ctx context.Context) (*DevopsWakapiProjectsResult, error) {
	var result DevopsWakapiProjectsResult
	if err := d.c.Get(ctx, "/service/devops/wakapi/projects", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DozzleHealth returns Dozzle health and version.
func (d *DevopsClient) DozzleHealth(ctx context.Context) (*capability.DevopsDozzleInfo, error) {
	var result DevopsDozzleInfoResult
	if err := d.c.Get(ctx, "/service/devops/dozzle/health", &result); err != nil {
		return nil, err
	}
	return &result.Info, nil
}

// NetalertxHealth checks NetAlertX reachability.
func (d *DevopsClient) NetalertxHealth(ctx context.Context) (bool, error) {
	var result DevopsHealthyResult
	if err := d.c.Get(ctx, "/service/devops/netalertx/health", &result); err != nil {
		return false, err
	}
	return result.Data["healthy"], nil
}

// NetalertxListDevices lists NetAlertX devices.
func (d *DevopsClient) NetalertxListDevices(ctx context.Context) (*DevopsNetalertxDevicesResult, error) {
	var result DevopsNetalertxDevicesResult
	if err := d.c.Get(ctx, "/service/devops/netalertx/devices", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// NetalertxTotals returns NetAlertX device category counts.
func (d *DevopsClient) NetalertxTotals(ctx context.Context) (*capability.DevopsNetalertxTotals, error) {
	var result DevopsNetalertxTotalsResult
	if err := d.c.Get(ctx, "/service/devops/netalertx/totals", &result); err != nil {
		return nil, err
	}
	return &result.Totals, nil
}

// NetalertxSearchDevices searches NetAlertX devices.
func (d *DevopsClient) NetalertxSearchDevices(ctx context.Context, query string) (*DevopsNetalertxDevicesResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	var result DevopsNetalertxDevicesResult
	if err := d.c.Post(ctx, "/service/devops/netalertx/devices/search", &DevopsNetalertxSearchRequest{Query: query}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
