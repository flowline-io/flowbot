// Package devops implements the multi-provider devops capability aggregator.
package devops

import (
	"context"
	"reflect"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/providers/beszel"
	"github.com/flowline-io/flowbot/pkg/providers/dozzle"
	"github.com/flowline-io/flowbot/pkg/providers/grafana"
	"github.com/flowline-io/flowbot/pkg/providers/netalertx"
	"github.com/flowline-io/flowbot/pkg/providers/scanopy"
	"github.com/flowline-io/flowbot/pkg/providers/traefik"
	"github.com/flowline-io/flowbot/pkg/providers/uptimekuma"
	"github.com/flowline-io/flowbot/pkg/providers/wakapi"
	"github.com/flowline-io/flowbot/pkg/types"
)

var scanopyCursorSecret = []byte("flowbot-capability-devops-scanopy-cursor-v1")

const scanopyCursorCap = "devops/scanopy"

type beszelClient interface {
	ListSystems(ctx context.Context) (*beszel.SystemList, error)
	GetSystem(ctx context.Context, id string) (*beszel.System, error)
}

type uptimekumaClient interface {
	Health(ctx context.Context) error
	Metrics(ctx context.Context) (map[string]*dto.MetricFamily, error)
}

type traefikClient interface {
	Overview(ctx context.Context) (*traefik.Overview, error)
	ListRouters(ctx context.Context) ([]traefik.Router, error)
	ListServices(ctx context.Context) ([]traefik.Service, error)
}

type grafanaClient interface {
	Health(ctx context.Context) (*grafana.Health, error)
	ListDatasources(ctx context.Context) ([]grafana.Datasource, error)
	SearchDashboards(ctx context.Context, query string) ([]grafana.DashboardHit, error)
	Query(ctx context.Context, req grafana.QueryRequest) (*grafana.QueryResult, error)
}

type wakapiClient interface {
	Health(ctx context.Context) (*wakapi.HealthStatus, error)
	GetStats(ctx context.Context, params wakapi.StatsParams) (*wakapi.Stats, error)
	GetAllTime(ctx context.Context, user string) (*wakapi.AllTime, error)
	ListProjects(ctx context.Context, query string) ([]wakapi.Project, error)
}

type dozzleClient interface {
	Health(ctx context.Context) error
	Version(ctx context.Context) (*dozzle.VersionInfo, error)
}

type netalertxClient interface {
	Health(ctx context.Context) error
	ListDevices(ctx context.Context) ([]netalertx.Device, error)
	GetTotals(ctx context.Context) (*netalertx.Totals, error)
	SearchDevices(ctx context.Context, query string) ([]netalertx.Device, error)
}

type scanopyClient interface {
	Health(ctx context.Context) error
	GetVersion(ctx context.Context) (*scanopy.VersionInfo, error)
	ListNetworks(ctx context.Context, params scanopy.ListParams) (*scanopy.Page[scanopy.Network], error)
	ListHosts(ctx context.Context, params scanopy.ListParams) (*scanopy.Page[scanopy.Host], error)
	GetHost(ctx context.Context, id string) (*scanopy.Host, error)
	ListServices(ctx context.Context, params scanopy.ListParams) (*scanopy.Page[scanopy.Service], error)
	ListDaemons(ctx context.Context, params scanopy.ListParams) (*scanopy.Page[scanopy.Daemon], error)
}

// Adapter implements Service using optional provider clients.
type Adapter struct {
	beszel     beszelClient
	uptimekuma uptimekumaClient
	traefik    traefikClient
	grafana    grafanaClient
	wakapi     wakapiClient
	dozzle     dozzleClient
	netalertx  netalertxClient
	scanopy    scanopyClient
	now        func() time.Time
}

// Clients holds optional provider clients for constructing an Adapter in tests.
type Clients struct {
	Beszel     beszelClient
	Uptimekuma uptimekumaClient
	Traefik    traefikClient
	Grafana    grafanaClient
	Wakapi     wakapiClient
	Dozzle     dozzleClient
	Netalertx  netalertxClient
	Scanopy    scanopyClient
}

// New creates an Adapter from configured providers.
// Returns nil when no devops provider is configured.
func New() Service {
	c := Clients{}
	// Assign only non-nil clients so interface fields stay true-nil
	// (a typed nil *T stored in an interface is not == nil).
	if v := beszel.GetClient(); v != nil {
		c.Beszel = v
	}
	if v := uptimekuma.GetClient(); v != nil {
		c.Uptimekuma = v
	}
	if v := traefik.GetClient(); v != nil {
		c.Traefik = v
	}
	if v := grafana.GetClient(); v != nil {
		c.Grafana = v
	}
	if v := wakapi.GetClient(); v != nil {
		c.Wakapi = v
	}
	if v := dozzle.GetClient(); v != nil {
		c.Dozzle = v
	}
	if v := netalertx.GetClient(); v != nil {
		c.Netalertx = v
	}
	if v := scanopy.GetClient(); v != nil {
		c.Scanopy = v
	}
	return NewWithClients(c)
}

// NewWithClients creates an Adapter from explicit clients (for tests).
// Returns nil when every client is nil.
func NewWithClients(c Clients) Service {
	c.Beszel = nilIfTypedNil(c.Beszel)
	c.Uptimekuma = nilIfTypedNil(c.Uptimekuma)
	c.Traefik = nilIfTypedNil(c.Traefik)
	c.Grafana = nilIfTypedNil(c.Grafana)
	c.Wakapi = nilIfTypedNil(c.Wakapi)
	c.Dozzle = nilIfTypedNil(c.Dozzle)
	c.Netalertx = nilIfTypedNil(c.Netalertx)
	c.Scanopy = nilIfTypedNil(c.Scanopy)

	if c.Beszel == nil && c.Uptimekuma == nil && c.Traefik == nil &&
		c.Grafana == nil && c.Wakapi == nil && c.Dozzle == nil && c.Netalertx == nil && c.Scanopy == nil {
		return nil
	}
	return &Adapter{
		beszel:     c.Beszel,
		uptimekuma: c.Uptimekuma,
		traefik:    c.Traefik,
		grafana:    c.Grafana,
		wakapi:     c.Wakapi,
		dozzle:     c.Dozzle,
		netalertx:  c.Netalertx,
		scanopy:    c.Scanopy,
		now:        time.Now,
	}
}

// nilIfTypedNil converts a typed-nil pointer held in an interface to a true nil interface.
func nilIfTypedNil[T any](v T) T {
	rv := reflect.ValueOf(&v).Elem()
	if !rv.IsValid() {
		return v
	}
	// T is an interface type (beszelClient, grafanaClient, …).
	if rv.Kind() != reflect.Interface {
		return v
	}
	if rv.IsNil() {
		var zero T
		return zero
	}
	elem := rv.Elem()
	switch elem.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		if elem.IsNil() {
			var zero T
			return zero
		}
	}
	return v
}

func notConfigured(id string) error {
	return types.WrapError(types.ErrProvider, id+" not configured", nil)
}

// Status reports which backends are configured.
func (a *Adapter) Status(ctx context.Context) (*capability.DevopsStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	return &capability.DevopsStatus{
		Backends: map[string]bool{
			"beszel":     a.beszel != nil,
			"uptimekuma": a.uptimekuma != nil,
			"traefik":    a.traefik != nil,
			"grafana":    a.grafana != nil,
			"wakapi":     a.wakapi != nil,
			"dozzle":     a.dozzle != nil,
			"netalertx":  a.netalertx != nil,
			"scanopy":    a.scanopy != nil,
		},
	}, nil
}

// HealthCheck probes configured backends that expose a health endpoint.
// Returns true when at least one such backend is configured and all of them succeed.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	probed := 0
	if a.uptimekuma != nil {
		probed++
		if err := a.uptimekuma.Health(ctx); err != nil {
			return false, wrapProviderErr("uptimekuma health check failed", err)
		}
	}
	if a.grafana != nil {
		probed++
		if _, err := a.grafana.Health(ctx); err != nil {
			return false, wrapProviderErr("grafana health check failed", err)
		}
	}
	if a.dozzle != nil {
		probed++
		if err := a.dozzle.Health(ctx); err != nil {
			return false, wrapProviderErr("dozzle health check failed", err)
		}
	}
	if a.netalertx != nil {
		probed++
		if err := a.netalertx.Health(ctx); err != nil {
			return false, wrapProviderErr("netalertx health check failed", err)
		}
	}
	if a.scanopy != nil {
		probed++
		if err := a.scanopy.Health(ctx); err != nil {
			return false, wrapProviderErr("scanopy health check failed", err)
		}
	}
	if a.wakapi != nil {
		probed++
		if err := probeWakapiHealth(ctx, a.wakapi); err != nil {
			return false, err
		}
	}
	if probed == 0 {
		return false, types.Errorf(types.ErrProvider, "no devops health backends configured")
	}
	return true, nil
}

func probeWakapiHealth(ctx context.Context, client wakapiClient) error {
	status, err := client.Health(ctx)
	if err != nil {
		return wrapProviderErr("wakapi health check failed", err)
	}
	if status != nil && (!status.AppOK || !status.DBOK) {
		return types.Errorf(types.ErrProvider, "wakapi unhealthy: app=%v db=%v", status.AppOK, status.DBOK)
	}
	return nil
}

// BeszelListSystems lists Beszel systems.
func (a *Adapter) BeszelListSystems(ctx context.Context) (*capability.ListResult[capability.DevopsSystem], error) {
	if a.beszel == nil {
		return nil, notConfigured("beszel")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	result, err := a.beszel.ListSystems(ctx)
	if err != nil {
		return nil, wrapProviderErr("beszel list systems failed", err)
	}
	items := make([]*capability.DevopsSystem, 0)
	if result != nil {
		items = make([]*capability.DevopsSystem, 0, len(result.Items))
		for _, s := range result.Items {
			items = append(items, &capability.DevopsSystem{
				ID: s.ID, Name: s.Name, Status: s.Status, Host: s.Host,
			})
		}
	}
	return &capability.ListResult[capability.DevopsSystem]{Items: items, Page: &capability.PageInfo{}}, nil
}

// BeszelGetSystem returns one Beszel system.
func (a *Adapter) BeszelGetSystem(ctx context.Context, in GetSystemInput) (*capability.DevopsSystem, error) {
	if a.beszel == nil {
		return nil, notConfigured("beszel")
	}
	if in.ID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	s, err := a.beszel.GetSystem(ctx, in.ID)
	if err != nil {
		return nil, wrapProviderErr("beszel get system failed", err)
	}
	if s == nil {
		return nil, types.Errorf(types.ErrNotFound, "system not found")
	}
	return &capability.DevopsSystem{ID: s.ID, Name: s.Name, Status: s.Status, Host: s.Host}, nil
}

// UptimekumaHealth checks Uptime Kuma metrics reachability.
func (a *Adapter) UptimekumaHealth(ctx context.Context) error {
	if a.uptimekuma == nil {
		return notConfigured("uptimekuma")
	}
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if err := a.uptimekuma.Health(ctx); err != nil {
		return wrapProviderErr("uptimekuma health failed", err)
	}
	return nil
}

// UptimekumaMetrics returns summarized Prometheus metric families.
func (a *Adapter) UptimekumaMetrics(ctx context.Context) (*capability.ListResult[capability.DevopsMetricFamily], error) {
	if a.uptimekuma == nil {
		return nil, notConfigured("uptimekuma")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	families, err := a.uptimekuma.Metrics(ctx)
	if err != nil {
		return nil, wrapProviderErr("uptimekuma metrics failed", err)
	}
	items := make([]*capability.DevopsMetricFamily, 0, len(families))
	for name, fam := range families {
		item := &capability.DevopsMetricFamily{Name: name}
		if fam != nil {
			if fam.Help != nil {
				item.Help = *fam.Help
			}
			item.Count = len(fam.Metric)
		}
		items = append(items, item)
	}
	return &capability.ListResult[capability.DevopsMetricFamily]{Items: items, Page: &capability.PageInfo{}}, nil
}

// TraefikOverview returns Traefik overview counts.
func (a *Adapter) TraefikOverview(ctx context.Context) (*capability.DevopsTraefikOverview, error) {
	if a.traefik == nil {
		return nil, notConfigured("traefik")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	ov, err := a.traefik.Overview(ctx)
	if err != nil {
		return nil, wrapProviderErr("traefik overview failed", err)
	}
	out := &capability.DevopsTraefikOverview{}
	if ov != nil && ov.HTTP != nil {
		out.HTTPRouters = ov.HTTP.Routers["total"]
		out.HTTPServices = ov.HTTP.Services["total"]
	}
	return out, nil
}

// TraefikListRouters lists Traefik HTTP routers.
func (a *Adapter) TraefikListRouters(ctx context.Context) (*capability.ListResult[capability.DevopsRouter], error) {
	if a.traefik == nil {
		return nil, notConfigured("traefik")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	routers, err := a.traefik.ListRouters(ctx)
	if err != nil {
		return nil, wrapProviderErr("traefik list routers failed", err)
	}
	items := make([]*capability.DevopsRouter, 0, len(routers))
	for _, r := range routers {
		items = append(items, &capability.DevopsRouter{
			Name: r.Name, Rule: r.Rule, Service: r.Service, Status: r.Status,
		})
	}
	return &capability.ListResult[capability.DevopsRouter]{Items: items, Page: &capability.PageInfo{}}, nil
}

// TraefikListServices lists Traefik HTTP services.
func (a *Adapter) TraefikListServices(ctx context.Context) (*capability.ListResult[capability.DevopsService], error) {
	if a.traefik == nil {
		return nil, notConfigured("traefik")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	services, err := a.traefik.ListServices(ctx)
	if err != nil {
		return nil, wrapProviderErr("traefik list services failed", err)
	}
	items := make([]*capability.DevopsService, 0, len(services))
	for _, s := range services {
		items = append(items, &capability.DevopsService{Name: s.Name, Type: s.Type, Status: s.Status})
	}
	return &capability.ListResult[capability.DevopsService]{Items: items, Page: &capability.PageInfo{}}, nil
}

// GrafanaHealth returns Grafana health.
func (a *Adapter) GrafanaHealth(ctx context.Context) (*capability.DevopsGrafanaHealth, error) {
	if a.grafana == nil {
		return nil, notConfigured("grafana")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	h, err := a.grafana.Health(ctx)
	if err != nil {
		return nil, wrapProviderErr("grafana health failed", err)
	}
	if h == nil {
		return &capability.DevopsGrafanaHealth{}, nil
	}
	return &capability.DevopsGrafanaHealth{Database: h.Database, Version: h.Version}, nil
}

// GrafanaListDatasources lists Grafana datasources.
func (a *Adapter) GrafanaListDatasources(ctx context.Context) (*capability.ListResult[capability.DevopsDatasource], error) {
	if a.grafana == nil {
		return nil, notConfigured("grafana")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.grafana.ListDatasources(ctx)
	if err != nil {
		return nil, wrapProviderErr("grafana list datasources failed", err)
	}
	items := make([]*capability.DevopsDatasource, 0, len(list))
	for _, d := range list {
		items = append(items, &capability.DevopsDatasource{ID: d.ID, UID: d.UID, Name: d.Name, Type: d.Type})
	}
	return &capability.ListResult[capability.DevopsDatasource]{Items: items, Page: &capability.PageInfo{}}, nil
}

// GrafanaSearchDashboards searches Grafana dashboards.
func (a *Adapter) GrafanaSearchDashboards(ctx context.Context, in SearchDashboardsInput) (*capability.ListResult[capability.DevopsDashboard], error) {
	if a.grafana == nil {
		return nil, notConfigured("grafana")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.grafana.SearchDashboards(ctx, in.Query)
	if err != nil {
		return nil, wrapProviderErr("grafana search dashboards failed", err)
	}
	items := make([]*capability.DevopsDashboard, 0, len(list))
	for _, d := range list {
		items = append(items, &capability.DevopsDashboard{UID: d.UID, Title: d.Title, URL: d.URL})
	}
	return &capability.ListResult[capability.DevopsDashboard]{Items: items, Page: &capability.PageInfo{}}, nil
}

// GrafanaQuery runs a query against prometheus/alloy/loki/tempo/pyroscope via Grafana.
func (a *Adapter) GrafanaQuery(ctx context.Context, in GrafanaQueryInput) (*capability.DevopsGrafanaQueryResult, error) {
	if a.grafana == nil {
		return nil, notConfigured("grafana")
	}
	if in.Backend == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "backend is required")
	}
	if in.Expr == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "expr is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	result, err := a.grafana.Query(ctx, grafana.QueryRequest{
		Backend:       grafana.BackendKind(in.Backend),
		Expr:          in.Expr,
		DatasourceUID: in.DatasourceUID,
		From:          in.From,
		To:            in.To,
		MaxLines:      in.MaxLines,
	})
	if err != nil {
		return nil, wrapProviderErr("grafana query failed", err)
	}
	out := &capability.DevopsGrafanaQueryResult{}
	if result != nil {
		out.Backend = string(result.Backend)
		out.DatasourceUID = result.DatasourceUID
		out.DatasourceType = result.DatasourceType
		for _, f := range result.Frames {
			frame := capability.DevopsGrafanaQueryFrame{Name: f.Name, RefID: f.RefID}
			for _, field := range f.Fields {
				frame.Fields = append(frame.Fields, capability.DevopsGrafanaQueryField{
					Name: field.Name, Type: field.Type, Values: field.Values,
				})
			}
			out.Frames = append(out.Frames, frame)
		}
	}
	return out, nil
}

// WakapiHealth checks Wakapi application health.
func (a *Adapter) WakapiHealth(ctx context.Context) (*capability.DevopsWakapiHealth, error) {
	if a.wakapi == nil {
		return nil, notConfigured("wakapi")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	status, err := a.wakapi.Health(ctx)
	if err != nil {
		return nil, wrapProviderErr("wakapi health failed", err)
	}
	out := &capability.DevopsWakapiHealth{}
	if status != nil {
		out.AppOK = status.AppOK
		out.DBOK = status.DBOK
	}
	return out, nil
}

// WakapiSummary returns a coding-stats summary.
func (a *Adapter) WakapiSummary(ctx context.Context, in SummaryInput) (*capability.DevopsWakapiSummary, error) {
	if a.wakapi == nil {
		return nil, notConfigured("wakapi")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	rangeVal := in.Interval
	if rangeVal == "" {
		rangeVal = "today"
	}
	stats, err := a.wakapi.GetStats(ctx, wakapi.StatsParams{
		Range:   rangeVal,
		Project: in.Project,
	})
	if err != nil {
		return nil, wrapProviderErr("wakapi summary failed", err)
	}
	out := &capability.DevopsWakapiSummary{}
	if stats != nil {
		out.TotalSeconds = int64(stats.TotalSeconds)
		out.HumanReadableTotal = stats.HumanReadableTotal
		out.HumanReadableRange = stats.HumanReadableRange
		out.Range = stats.Range
	}
	return out, nil
}

// WakapiAllTime returns cumulative coding time since account creation.
func (a *Adapter) WakapiAllTime(ctx context.Context) (*capability.DevopsWakapiAllTime, error) {
	if a.wakapi == nil {
		return nil, notConfigured("wakapi")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	allTime, err := a.wakapi.GetAllTime(ctx, wakapi.CurrentUser)
	if err != nil {
		return nil, wrapProviderErr("wakapi all time failed", err)
	}
	out := &capability.DevopsWakapiAllTime{}
	if allTime != nil {
		out.TotalSeconds = int64(allTime.TotalSeconds)
		out.Text = allTime.Text
		out.IsUpToDate = allTime.IsUpToDate
	}
	return out, nil
}

// WakapiListProjects lists Wakapi projects.
func (a *Adapter) WakapiListProjects(ctx context.Context, in ListProjectsInput) (*capability.ListResult[capability.DevopsWakapiProject], error) {
	if a.wakapi == nil {
		return nil, notConfigured("wakapi")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.wakapi.ListProjects(ctx, in.Query)
	if err != nil {
		return nil, wrapProviderErr("wakapi list projects failed", err)
	}
	items := make([]*capability.DevopsWakapiProject, 0, len(list))
	for _, p := range list {
		items = append(items, &capability.DevopsWakapiProject{ID: p.ID, Name: p.Name})
	}
	return &capability.ListResult[capability.DevopsWakapiProject]{Items: items, Page: &capability.PageInfo{}}, nil
}

// DozzleHealth checks Dozzle and returns version when available.
func (a *Adapter) DozzleHealth(ctx context.Context) (*capability.DevopsDozzleInfo, error) {
	if a.dozzle == nil {
		return nil, notConfigured("dozzle")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if err := a.dozzle.Health(ctx); err != nil {
		return nil, wrapProviderErr("dozzle health failed", err)
	}
	info := &capability.DevopsDozzleInfo{Healthy: true}
	if ver, err := a.dozzle.Version(ctx); err == nil && ver != nil {
		info.Version = ver.Version
	}
	return info, nil
}

// NetalertxHealth checks NetAlertX reachability.
func (a *Adapter) NetalertxHealth(ctx context.Context) error {
	if a.netalertx == nil {
		return notConfigured("netalertx")
	}
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if err := a.netalertx.Health(ctx); err != nil {
		return wrapProviderErr("netalertx health failed", err)
	}
	return nil
}

// NetalertxListDevices lists NetAlertX devices.
func (a *Adapter) NetalertxListDevices(ctx context.Context) (*capability.ListResult[capability.DevopsNetalertxDevice], error) {
	if a.netalertx == nil {
		return nil, notConfigured("netalertx")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.netalertx.ListDevices(ctx)
	if err != nil {
		return nil, wrapProviderErr("netalertx list devices failed", err)
	}
	items := make([]*capability.DevopsNetalertxDevice, 0, len(list))
	for _, d := range list {
		items = append(items, toNetalertxDevice(d))
	}
	return &capability.ListResult[capability.DevopsNetalertxDevice]{Items: items, Page: &capability.PageInfo{}}, nil
}

// NetalertxTotals returns NetAlertX device category counts.
func (a *Adapter) NetalertxTotals(ctx context.Context) (*capability.DevopsNetalertxTotals, error) {
	if a.netalertx == nil {
		return nil, notConfigured("netalertx")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	t, err := a.netalertx.GetTotals(ctx)
	if err != nil {
		return nil, wrapProviderErr("netalertx totals failed", err)
	}
	out := &capability.DevopsNetalertxTotals{}
	if t != nil {
		out.All = t.All
		out.Connected = t.Connected
		out.Favorites = t.Favorites
		out.New = t.New
		out.Down = t.Down
		out.Archived = t.Archived
	}
	return out, nil
}

// NetalertxSearchDevices searches NetAlertX devices.
func (a *Adapter) NetalertxSearchDevices(ctx context.Context, in SearchDevicesInput) (*capability.ListResult[capability.DevopsNetalertxDevice], error) {
	if a.netalertx == nil {
		return nil, notConfigured("netalertx")
	}
	if in.Query == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "query is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.netalertx.SearchDevices(ctx, in.Query)
	if err != nil {
		return nil, wrapProviderErr("netalertx search devices failed", err)
	}
	items := make([]*capability.DevopsNetalertxDevice, 0, len(list))
	for _, d := range list {
		items = append(items, toNetalertxDevice(d))
	}
	return &capability.ListResult[capability.DevopsNetalertxDevice]{Items: items, Page: &capability.PageInfo{}}, nil
}

func toNetalertxDevice(d netalertx.Device) *capability.DevopsNetalertxDevice {
	mac := d.MAC
	if mac == "" {
		mac = d.MacAlt
	}
	ip := d.IP
	if ip == "" {
		ip = d.LastIP
	}
	return &capability.DevopsNetalertxDevice{
		Name: d.Name, MAC: mac, IP: ip, Type: d.Type, Status: d.Status, Vendor: d.Vendor,
	}
}

func toScanopyListParams(in ScanopyListInput, offset, limit int) scanopy.ListParams {
	return scanopy.ListParams{
		NetworkID: in.NetworkID,
		HostID:    in.HostID,
		Search:    in.Search,
		Limit:     new(limit),
		Offset:    offset,
	}
}

func (a *Adapter) scanopyOffsetLimit(in ScanopyListInput) (offset, limit int) {
	limit = in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	offset = 0
	if in.Cursor == "" {
		return offset, limit
	}
	now := time.Now
	if a.now != nil {
		now = a.now
	}
	payload, err := capability.DecodeCursor(scanopyCursorSecret, in.Cursor, now())
	if err != nil || payload.Capability != scanopyCursorCap {
		return offset, limit
	}
	offset = payload.Offset
	if payload.Limit > 0 {
		limit = min(payload.Limit, 1000)
	}
	return offset, limit
}

func scanopyPageInfo(pag scanopy.PaginationMeta, itemCount int) *capability.PageInfo {
	limit := pag.Limit
	if limit <= 0 {
		limit = itemCount
	}
	page := &capability.PageInfo{Limit: limit, HasMore: pag.HasMore}
	if pag.TotalCount > 0 {
		total := pag.TotalCount
		page.Total = &total
	}
	if !pag.HasMore {
		return page
	}
	nextOffset := pag.Offset + itemCount
	if itemCount == 0 && limit > 0 {
		nextOffset = pag.Offset + limit
	}
	cursor, err := capability.EncodeCursor(scanopyCursorSecret, capability.CursorPayload{
		Capability: scanopyCursorCap,
		Strategy:   "offset",
		Offset:     nextOffset,
		Limit:      limit,
	})
	if err == nil {
		page.NextCursor = cursor
	}
	return page
}

func toScanopyHost(h scanopy.Host) *capability.DevopsScanopyHost {
	ips := make([]string, 0, len(h.IPAddresses))
	for _, ip := range h.IPAddresses {
		if ip.IPAddress != "" {
			ips = append(ips, ip.IPAddress)
		}
	}
	return &capability.DevopsScanopyHost{
		ID: h.ID, Name: h.Name, Hostname: h.Hostname, NetworkID: h.NetworkID,
		LastSeenAt: h.LastSeenAt, IPs: ips, Hidden: h.Hidden,
	}
}

// ScanopyHealth checks Scanopy reachability.
func (a *Adapter) ScanopyHealth(ctx context.Context) error {
	if a.scanopy == nil {
		return notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if err := a.scanopy.Health(ctx); err != nil {
		return wrapProviderErr("scanopy health failed", err)
	}
	return nil
}

// ScanopyVersion returns Scanopy API and server versions.
func (a *Adapter) ScanopyVersion(ctx context.Context) (*capability.DevopsScanopyVersion, error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	info, err := a.scanopy.GetVersion(ctx)
	if err != nil {
		return nil, wrapProviderErr("scanopy version failed", err)
	}
	out := &capability.DevopsScanopyVersion{}
	if info != nil {
		out.APIVersion = info.APIVersion
		out.ServerVersion = info.ServerVersion
	}
	return out, nil
}

// ScanopyListNetworks lists Scanopy networks.
func (a *Adapter) ScanopyListNetworks(ctx context.Context, in ScanopyListInput) (*capability.ListResult[capability.DevopsScanopyNetwork], error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	offset, limit := a.scanopyOffsetLimit(in)
	page, err := a.scanopy.ListNetworks(ctx, toScanopyListParams(in, offset, limit))
	if err != nil {
		return nil, wrapProviderErr("scanopy list networks failed", err)
	}
	if page == nil {
		page = &scanopy.Page[scanopy.Network]{}
	}
	items := make([]*capability.DevopsScanopyNetwork, 0, len(page.Items))
	for _, n := range page.Items {
		items = append(items, &capability.DevopsScanopyNetwork{
			ID: n.ID, Name: n.Name, OrganizationID: n.OrganizationID,
		})
	}
	return &capability.ListResult[capability.DevopsScanopyNetwork]{
		Items: items,
		Page:  scanopyPageInfo(page.Pagination, len(items)),
	}, nil
}

// ScanopyListHosts lists Scanopy hosts.
func (a *Adapter) ScanopyListHosts(ctx context.Context, in ScanopyListInput) (*capability.ListResult[capability.DevopsScanopyHost], error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	offset, limit := a.scanopyOffsetLimit(in)
	page, err := a.scanopy.ListHosts(ctx, toScanopyListParams(in, offset, limit))
	if err != nil {
		return nil, wrapProviderErr("scanopy list hosts failed", err)
	}
	if page == nil {
		page = &scanopy.Page[scanopy.Host]{}
	}
	items := make([]*capability.DevopsScanopyHost, 0, len(page.Items))
	for _, h := range page.Items {
		items = append(items, toScanopyHost(h))
	}
	return &capability.ListResult[capability.DevopsScanopyHost]{
		Items: items,
		Page:  scanopyPageInfo(page.Pagination, len(items)),
	}, nil
}

// ScanopyGetHost returns a Scanopy host by ID.
func (a *Adapter) ScanopyGetHost(ctx context.Context, in ScanopyGetHostInput) (*capability.DevopsScanopyHost, error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if in.ID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	h, err := a.scanopy.GetHost(ctx, in.ID)
	if err != nil {
		return nil, wrapProviderErr("scanopy get host failed", err)
	}
	if h == nil {
		return &capability.DevopsScanopyHost{}, nil
	}
	return toScanopyHost(*h), nil
}

// ScanopyListServices lists Scanopy services.
func (a *Adapter) ScanopyListServices(ctx context.Context, in ScanopyListInput) (*capability.ListResult[capability.DevopsScanopyService], error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	offset, limit := a.scanopyOffsetLimit(in)
	page, err := a.scanopy.ListServices(ctx, toScanopyListParams(in, offset, limit))
	if err != nil {
		return nil, wrapProviderErr("scanopy list services failed", err)
	}
	if page == nil {
		page = &scanopy.Page[scanopy.Service]{}
	}
	items := make([]*capability.DevopsScanopyService, 0, len(page.Items))
	for _, s := range page.Items {
		items = append(items, &capability.DevopsScanopyService{
			ID: s.ID, Name: s.Name, HostID: s.HostID, NetworkID: s.NetworkID, ServiceDefinition: s.ServiceDefinition,
		})
	}
	return &capability.ListResult[capability.DevopsScanopyService]{
		Items: items,
		Page:  scanopyPageInfo(page.Pagination, len(items)),
	}, nil
}

// ScanopyListDaemons lists Scanopy daemons.
func (a *Adapter) ScanopyListDaemons(ctx context.Context, in ScanopyListInput) (*capability.ListResult[capability.DevopsScanopyDaemon], error) {
	if a.scanopy == nil {
		return nil, notConfigured("scanopy")
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	offset, limit := a.scanopyOffsetLimit(in)
	page, err := a.scanopy.ListDaemons(ctx, toScanopyListParams(in, offset, limit))
	if err != nil {
		return nil, wrapProviderErr("scanopy list daemons failed", err)
	}
	if page == nil {
		page = &scanopy.Page[scanopy.Daemon]{}
	}
	items := make([]*capability.DevopsScanopyDaemon, 0, len(page.Items))
	for _, d := range page.Items {
		items = append(items, &capability.DevopsScanopyDaemon{
			ID: d.ID, Name: d.Name, NetworkID: d.NetworkID, Version: d.Version,
			IsUnreachable: d.IsUnreachable, LastSeen: d.LastSeen,
		})
	}
	return &capability.ListResult[capability.DevopsScanopyDaemon]{
		Items: items,
		Page:  scanopyPageInfo(page.Pagination, len(items)),
	}, nil
}

func wrapProviderErr(msg string, err error) error {
	return types.WrapError(types.ErrProvider, msg, err)
}

// Ensure Adapter implements Service.
var _ Service = (*Adapter)(nil)
