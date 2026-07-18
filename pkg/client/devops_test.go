package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevopsStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantBeszel bool
		wantErr    bool
	}{
		{
			name: "status success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/devops/status", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"backends":{"beszel":true,"grafana":false}}}}`))
			},
			wantBeszel: true,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"server error"}`))
			},
			wantErr: true,
		},
		{
			name: "empty backends",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"backends":{}}}}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			got, err := NewClient(server.URL, "token").Devops.Status(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBeszel, got.Backends["beszel"])
		})
	}
}

func TestDevopsBeszelAndGrafana(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fn      func(*Client) error
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "list systems",
			fn: func(c *Client) error {
				got, err := c.Devops.BeszelListSystems(context.Background())
				if err != nil {
					return err
				}
				if len(got.Items) != 1 || got.Items[0].Name != "host-a" {
					return assert.AnError
				}
				return nil
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/devops/beszel/systems", r.URL.Path)
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"1","name":"host-a","status":"up"}]}}`))
			},
		},
		{
			name: "get system requires id",
			fn: func(c *Client) error {
				_, err := c.Devops.BeszelGetSystem(context.Background(), "")
				return err
			},
			handler: func(_ http.ResponseWriter, _ *http.Request) {},
			wantErr: true,
		},
		{
			name: "grafana health",
			fn: func(c *Client) error {
				got, err := c.Devops.GrafanaHealth(context.Background())
				if err != nil {
					return err
				}
				if got.Version != "11.0.0" {
					return assert.AnError
				}
				return nil
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/devops/grafana/health", r.URL.Path)
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"database":"ok","version":"11.0.0"}}}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			err := tt.fn(NewClient(server.URL, "token"))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDevopsDozzleHealth(t *testing.T) {
	t.Parallel()
	t.Run("healthy", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/service/devops/dozzle/health", r.URL.Path)
			_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"healthy":true,"version":"v8"}}}`))
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "token").Devops.DozzleHealth(context.Background())
		require.NoError(t, err)
		assert.True(t, got.Healthy)
		assert.Equal(t, "v8", got.Version)
	})
	t.Run("grafana query", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/service/devops/grafana/query", r.URL.Path)
			_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"backend":"prometheus","datasource_uid":"p1","datasource_type":"prometheus","frames":[]}}}`))
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "token").Devops.GrafanaQuery(context.Background(), DevopsGrafanaQueryRequest{
			Backend: "prometheus", Expr: "up",
		})
		require.NoError(t, err)
		assert.Equal(t, "prometheus", got.Backend)
	})
	t.Run("netalertx devices", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/service/devops/netalertx/devices", r.URL.Path)
			_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"name":"Router","mac":"AA:BB","status":"online"}]}}`))
		}))
		defer server.Close()
		got, err := NewClient(server.URL, "token").Devops.NetalertxListDevices(context.Background())
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "Router", got.Items[0].Name)
	})
}
