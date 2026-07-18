package devops

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the devops capability with hub and invoker registry.
// When svc is nil no devops provider is configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapDevops,
		App:         app,
		Description: "DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, and dozzle",
		Instance:    svc,
		Ops: []capability.OpDef{
			{Name: OpHealth, Description: "Aggregate health of configured devops backends", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeHealth(svc)},
			{Name: OpStatus, Description: "Configured devops backends", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeStatus(svc)},
			{Name: OpBeszelListSystems, Description: "List Beszel systems", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeBeszelListSystems(svc)},
			{
				Name: OpBeszelGetSystem, Description: "Get a Beszel system", Scopes: []string{auth.ScopeServiceDevopsRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "System ID"}},
				Handler: invokeBeszelGetSystem(svc),
			},
			{Name: OpUptimekumaHealth, Description: "Uptime Kuma health", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeUptimekumaHealth(svc)},
			{Name: OpUptimekumaMetrics, Description: "Uptime Kuma metrics summary", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeUptimekumaMetrics(svc)},
			{Name: OpTraefikOverview, Description: "Traefik overview", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeTraefikOverview(svc)},
			{Name: OpTraefikListRouters, Description: "List Traefik routers", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeTraefikListRouters(svc)},
			{Name: OpTraefikListServices, Description: "List Traefik services", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeTraefikListServices(svc)},
			{Name: OpGrafanaHealth, Description: "Grafana health", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeGrafanaHealth(svc)},
			{Name: OpGrafanaListDatasources, Description: "List Grafana datasources", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeGrafanaListDatasources(svc)},
			{
				Name: OpGrafanaSearchDashboards, Description: "Search Grafana dashboards", Scopes: []string{auth.ScopeServiceDevopsRead},
				Input:   []hub.ParamDef{{Name: "query", Type: "string", Description: "Search query"}},
				Handler: invokeGrafanaSearchDashboards(svc),
			},
			{
				Name: OpGrafanaQuery, Description: "Query prometheus/alloy/loki/tempo/pyroscope via Grafana", Scopes: []string{auth.ScopeServiceDevopsRead},
				Input: []hub.ParamDef{
					{Name: "backend", Type: "string", Required: true, Description: "prometheus|alloy|loki|tempo|pyroscope"},
					{Name: "expr", Type: "string", Required: true, Description: "Query expression"},
					{Name: "datasource_uid", Type: "string", Description: "Optional datasource UID"},
					{Name: "from", Type: "string", Description: "Grafana from time (default now-1h)"},
					{Name: "to", Type: "string", Description: "Grafana to time (default now)"},
					{Name: "max_lines", Type: "number", Description: "Loki max lines"},
				},
				Handler: invokeGrafanaQuery(svc),
			},
			{
				Name: OpWakapiSummary, Description: "Wakapi activity summary", Scopes: []string{auth.ScopeServiceDevopsRead},
				Input:   []hub.ParamDef{{Name: "interval", Type: "string", Description: "Interval (default today)"}},
				Handler: invokeWakapiSummary(svc),
			},
			{Name: OpWakapiListProjects, Description: "List Wakapi projects", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeWakapiListProjects(svc)},
			{Name: OpDozzleHealth, Description: "Dozzle health and version", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeDozzleHealth(svc)},
			{Name: OpNetalertxHealth, Description: "NetAlertX health", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeNetalertxHealth(svc)},
			{Name: OpNetalertxListDevices, Description: "List NetAlertX devices", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeNetalertxListDevices(svc)},
			{Name: OpNetalertxTotals, Description: "NetAlertX device totals", Scopes: []string{auth.ScopeServiceDevopsRead}, Handler: invokeNetalertxTotals(svc)},
			{
				Name: OpNetalertxSearchDevices, Description: "Search NetAlertX devices", Scopes: []string{auth.ScopeServiceDevopsRead},
				Input:   []hub.ParamDef{{Name: "query", Type: "string", Required: true, Description: "Search query"}},
				Handler: invokeNetalertxSearchDevices(svc),
			},
		},
	})
}

func invokeStatus(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		data, err := svc.Status(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok}, nil
	}
}

func invokeBeszelListSystems(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.BeszelListSystems(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsSystem]{Items: []*capability.DevopsSystem{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeBeszelGetSystem(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.BeszelGetSystem(ctx, GetSystemInput{ID: id})
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeUptimekumaHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		if err := svc.UptimekumaHealth(ctx); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]bool{"healthy": true}}, nil
	}
}

func invokeUptimekumaMetrics(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.UptimekumaMetrics(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsMetricFamily]{Items: []*capability.DevopsMetricFamily{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeTraefikOverview(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		data, err := svc.TraefikOverview(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeTraefikListRouters(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.TraefikListRouters(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsRouter]{Items: []*capability.DevopsRouter{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeTraefikListServices(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.TraefikListServices(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsService]{Items: []*capability.DevopsService{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGrafanaHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		data, err := svc.GrafanaHealth(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeGrafanaListDatasources(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.GrafanaListDatasources(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsDatasource]{Items: []*capability.DevopsDatasource{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGrafanaSearchDashboards(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		in := SearchDashboardsInput{}
		if q, ok := capability.StringParam(params, "query"); ok {
			in.Query = q
		}
		result, err := svc.GrafanaSearchDashboards(ctx, in)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsDashboard]{Items: []*capability.DevopsDashboard{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGrafanaQuery(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		backend, err := capability.RequiredString(params, "backend")
		if err != nil {
			return nil, err
		}
		expr, err := capability.RequiredString(params, "expr")
		if err != nil {
			return nil, err
		}
		in := GrafanaQueryInput{Backend: backend, Expr: expr}
		if uid, ok := capability.StringParam(params, "datasource_uid"); ok {
			in.DatasourceUID = uid
		}
		if from, ok := capability.StringParam(params, "from"); ok {
			in.From = from
		}
		if to, ok := capability.StringParam(params, "to"); ok {
			in.To = to
		}
		if maxLines, ok := capability.IntParam(params, "max_lines"); ok {
			in.MaxLines = maxLines
		}
		data, err := svc.GrafanaQuery(ctx, in)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeWakapiSummary(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		in := SummaryInput{}
		if interval, ok := capability.StringParam(params, "interval"); ok {
			in.Interval = interval
		}
		data, err := svc.WakapiSummary(ctx, in)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeWakapiListProjects(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.WakapiListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsWakapiProject]{Items: []*capability.DevopsWakapiProject{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeDozzleHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		data, err := svc.DozzleHealth(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeNetalertxHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		if err := svc.NetalertxHealth(ctx); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]bool{"healthy": true}}, nil
	}
}

func invokeNetalertxListDevices(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.NetalertxListDevices(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsNetalertxDevice]{Items: []*capability.DevopsNetalertxDevice{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeNetalertxTotals(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		data, err := svc.NetalertxTotals(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: data}, nil
	}
}

func invokeNetalertxSearchDevices(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		query, err := capability.RequiredString(params, "query")
		if err != nil {
			return nil, err
		}
		result, err := svc.NetalertxSearchDevices(ctx, SearchDevicesInput{Query: query})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.DevopsNetalertxDevice]{Items: []*capability.DevopsNetalertxDevice{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}
