package devops

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// GetSystemInput holds parameters for fetching a Beszel system.
type GetSystemInput struct {
	ID string
}

// SearchDashboardsInput holds parameters for Grafana dashboard search.
type SearchDashboardsInput struct {
	Query string
}

// GrafanaQueryInput holds parameters for a Grafana observability backend query.
type GrafanaQueryInput struct {
	Backend       string
	Expr          string
	DatasourceUID string
	From          string
	To            string
	MaxLines      int
}

// SummaryInput holds parameters for Wakapi summary.
type SummaryInput struct {
	Interval string
}

// SearchDevicesInput holds parameters for NetAlertX device search.
type SearchDevicesInput struct {
	Query string
}

// Service defines the devops aggregator capability contract.
type Service interface {
	Status(ctx context.Context) (*capability.DevopsStatus, error)

	BeszelListSystems(ctx context.Context) (*capability.ListResult[capability.DevopsSystem], error)
	BeszelGetSystem(ctx context.Context, in GetSystemInput) (*capability.DevopsSystem, error)

	UptimekumaHealth(ctx context.Context) error
	UptimekumaMetrics(ctx context.Context) (*capability.ListResult[capability.DevopsMetricFamily], error)

	TraefikOverview(ctx context.Context) (*capability.DevopsTraefikOverview, error)
	TraefikListRouters(ctx context.Context) (*capability.ListResult[capability.DevopsRouter], error)
	TraefikListServices(ctx context.Context) (*capability.ListResult[capability.DevopsService], error)

	GrafanaHealth(ctx context.Context) (*capability.DevopsGrafanaHealth, error)
	GrafanaListDatasources(ctx context.Context) (*capability.ListResult[capability.DevopsDatasource], error)
	GrafanaSearchDashboards(ctx context.Context, in SearchDashboardsInput) (*capability.ListResult[capability.DevopsDashboard], error)
	GrafanaQuery(ctx context.Context, in GrafanaQueryInput) (*capability.DevopsGrafanaQueryResult, error)

	WakapiSummary(ctx context.Context, in SummaryInput) (*capability.DevopsWakapiSummary, error)
	WakapiListProjects(ctx context.Context) (*capability.ListResult[capability.DevopsWakapiProject], error)

	DozzleHealth(ctx context.Context) (*capability.DevopsDozzleInfo, error)

	NetalertxHealth(ctx context.Context) error
	NetalertxListDevices(ctx context.Context) (*capability.ListResult[capability.DevopsNetalertxDevice], error)
	NetalertxTotals(ctx context.Context) (*capability.DevopsNetalertxTotals, error)
	NetalertxSearchDevices(ctx context.Context, in SearchDevicesInput) (*capability.ListResult[capability.DevopsNetalertxDevice], error)
}
