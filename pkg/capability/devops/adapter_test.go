package devops

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers/beszel"
	"github.com/flowline-io/flowbot/pkg/providers/dozzle"
	"github.com/flowline-io/flowbot/pkg/providers/grafana"
	"github.com/flowline-io/flowbot/pkg/providers/netalertx"
	"github.com/flowline-io/flowbot/pkg/providers/traefik"
	"github.com/flowline-io/flowbot/pkg/types"
)

type stubBeszel struct {
	systems *beszel.SystemList
	system  *beszel.System
	err     error
}

func (s *stubBeszel) ListSystems(context.Context) (*beszel.SystemList, error) {
	return s.systems, s.err
}
func (s *stubBeszel) GetSystem(context.Context, string) (*beszel.System, error) {
	return s.system, s.err
}

type stubUptime struct {
	err     error
	metrics map[string]*dto.MetricFamily
}

func (s *stubUptime) Health(context.Context) error { return s.err }
func (s *stubUptime) Metrics(context.Context) (map[string]*dto.MetricFamily, error) {
	return s.metrics, s.err
}

type stubTraefik struct {
	overview *traefik.Overview
	routers  []traefik.Router
	services []traefik.Service
	err      error
}

func (s *stubTraefik) Overview(context.Context) (*traefik.Overview, error) { return s.overview, s.err }
func (s *stubTraefik) ListRouters(context.Context) ([]traefik.Router, error) {
	return s.routers, s.err
}
func (s *stubTraefik) ListServices(context.Context) ([]traefik.Service, error) {
	return s.services, s.err
}

type stubGrafana struct {
	health *grafana.Health
	ds     []grafana.Datasource
	dash   []grafana.DashboardHit
	query  *grafana.QueryResult
	err    error
}

func (s *stubGrafana) Health(context.Context) (*grafana.Health, error) { return s.health, s.err }
func (s *stubGrafana) ListDatasources(context.Context) ([]grafana.Datasource, error) {
	return s.ds, s.err
}
func (s *stubGrafana) SearchDashboards(context.Context, string) ([]grafana.DashboardHit, error) {
	return s.dash, s.err
}
func (s *stubGrafana) Query(_ context.Context, _ grafana.QueryRequest) (*grafana.QueryResult, error) {
	return s.query, s.err
}

type stubDozzle struct {
	err     error
	version *dozzle.VersionInfo
}

func (s *stubDozzle) Health(context.Context) error { return s.err }
func (s *stubDozzle) Version(context.Context) (*dozzle.VersionInfo, error) {
	return s.version, s.err
}

type stubNetalertx struct {
	devices []netalertx.Device
	totals  *netalertx.Totals
	err     error
}

func (s *stubNetalertx) Health(context.Context) error { return s.err }
func (s *stubNetalertx) ListDevices(context.Context) ([]netalertx.Device, error) {
	return s.devices, s.err
}
func (s *stubNetalertx) GetTotals(context.Context) (*netalertx.Totals, error) {
	return s.totals, s.err
}
func (s *stubNetalertx) SearchDevices(context.Context, string) ([]netalertx.Device, error) {
	return s.devices, s.err
}

func TestNewWithClients(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		clients Clients
		wantNil bool
	}{
		{name: "all nil", clients: Clients{}, wantNil: true},
		{name: "beszel only", clients: Clients{Beszel: &stubBeszel{}}, wantNil: false},
		{name: "dozzle only", clients: Clients{Dozzle: &stubDozzle{}}, wantNil: false},
		{name: "netalertx only", clients: Clients{Netalertx: &stubNetalertx{}}, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewWithClients(tt.clients)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestAdapter_StatusAndNotConfigured(t *testing.T) {
	t.Parallel()
	t.Run("status reports configured backends", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Beszel: &stubBeszel{}, Grafana: &stubGrafana{}})
		st, err := svc.Status(context.Background())
		require.NoError(t, err)
		assert.True(t, st.Backends["beszel"])
		assert.True(t, st.Backends["grafana"])
		assert.False(t, st.Backends["traefik"])
	})
	t.Run("beszel not configured", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Grafana: &stubGrafana{}})
		_, err := svc.BeszelListSystems(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrProvider)
	})
	t.Run("traefik list routers", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Traefik: &stubTraefik{
			routers: []traefik.Router{{Name: "web@docker", Rule: "Host(`x`)", Status: "enabled"}},
		}})
		got, err := svc.TraefikListRouters(context.Background())
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "web@docker", got.Items[0].Name)
	})
}

func TestAdapter_BeszelAndGrafana(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		run     func(Service) error
		wantErr bool
	}{
		{
			name: "list systems",
			run: func(svc Service) error {
				got, err := svc.BeszelListSystems(context.Background())
				if err != nil {
					return err
				}
				if len(got.Items) != 1 || got.Items[0].ID != "s1" {
					return assert.AnError
				}
				return nil
			},
		},
		{
			name: "get system requires id",
			run: func(svc Service) error {
				_, err := svc.BeszelGetSystem(context.Background(), GetSystemInput{})
				return err
			},
			wantErr: true,
		},
		{
			name: "grafana health",
			run: func(svc Service) error {
				got, err := svc.GrafanaHealth(context.Background())
				if err != nil {
					return err
				}
				if got.Version != "11.0.0" {
					return assert.AnError
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClients(Clients{
				Beszel: &stubBeszel{
					systems: &beszel.SystemList{Items: []beszel.System{{ID: "s1", Name: "host"}}},
					system:  &beszel.System{ID: "s1", Name: "host"},
				},
				Grafana: &stubGrafana{health: &grafana.Health{Version: "11.0.0", Database: "ok"}},
			})
			err := tt.run(svc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdapter_DozzleHealth(t *testing.T) {
	t.Parallel()
	t.Run("healthy with version", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Dozzle: &stubDozzle{version: &dozzle.VersionInfo{Version: "v8"}}})
		got, err := svc.DozzleHealth(context.Background())
		require.NoError(t, err)
		assert.True(t, got.Healthy)
		assert.Equal(t, "v8", got.Version)
	})
	t.Run("health error", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Dozzle: &stubDozzle{err: assert.AnError}})
		_, err := svc.DozzleHealth(context.Background())
		assert.Error(t, err)
	})
	t.Run("grafana query", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Grafana: &stubGrafana{
			query: &grafana.QueryResult{Backend: grafana.BackendPrometheus, DatasourceUID: "p1"},
		}})
		got, err := svc.GrafanaQuery(context.Background(), GrafanaQueryInput{Backend: "prometheus", Expr: "up"})
		require.NoError(t, err)
		assert.Equal(t, "prometheus", got.Backend)
		assert.Equal(t, "p1", got.DatasourceUID)
	})
	t.Run("netalertx list devices", func(t *testing.T) {
		t.Parallel()
		svc := NewWithClients(Clients{Netalertx: &stubNetalertx{
			devices: []netalertx.Device{{Name: "Router", MAC: "AA:BB", Status: "online"}},
		}})
		got, err := svc.NetalertxListDevices(context.Background())
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "Router", got.Items[0].Name)
	})
}
