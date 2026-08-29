package scanopy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanopy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:60072", wantNil: false},
		{name: "trailing slash", endpoint: "http://localhost:60072/", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewScanopy(tt.endpoint, "scp_u_test")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestScanopy_GetVersionAndHealth(t *testing.T) {
	t.Parallel()
	t.Run("version envelope", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/version", r.URL.Path)
			assert.Equal(t, "Bearer scp_u_test", r.Header.Get("Authorization"))
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[VersionInfo]{
				Success: true,
				Data:    VersionInfo{APIVersion: 1, ServerVersion: "0.17.13"},
				Meta:    Meta{APIVersion: 1, ServerVersion: "0.17.13"},
			})
		}))
		defer server.Close()
		client := NewScanopy(server.URL, "scp_u_test")
		got, err := client.GetVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, got.APIVersion)
		assert.Equal(t, "0.17.13", got.ServerVersion)
		assert.NoError(t, client.Health(context.Background()))
	})
	t.Run("version direct", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(VersionInfo{APIVersion: 1, ServerVersion: "0.17.0"})
		}))
		defer server.Close()
		got, err := NewScanopy(server.URL, "k").GetVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "0.17.0", got.ServerVersion)
	})
	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[any]{Success: false, Error: "unauthorized"})
		}))
		defer server.Close()
		_, err := NewScanopy(server.URL, "bad").GetVersion(context.Background())
		assert.Error(t, err)
	})
	t.Run("nil client", func(t *testing.T) {
		t.Parallel()
		var client *Scanopy
		assert.Error(t, client.Health(context.Background()))
	})
}

func TestScanopy_ListHosts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hosts", r.URL.Path)
		assert.Equal(t, "net-1", r.URL.Query().Get("network_id"))
		assert.Equal(t, "router", r.URL.Query().Get("search"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[[]Host]{
			Success: true,
			Data: []Host{{
				ID: "h1", Name: "router", NetworkID: "net-1",
				IPAddresses: []IPAddress{{ID: "ip1", IPAddress: "192.168.1.1"}},
			}},
			Meta: Meta{
				APIVersion: 1, ServerVersion: "0.17.13",
				Pagination: &PaginationMeta{TotalCount: 1, Limit: 10, Offset: 0, HasMore: false},
			},
		})
	}))
	defer server.Close()
	page, err := NewScanopy(server.URL, "tok").ListHosts(context.Background(), ListParams{
		NetworkID: "net-1", Search: "router", Limit: LimitOf(10),
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "router", page.Items[0].Name)
	assert.Equal(t, "192.168.1.1", page.Items[0].IPAddresses[0].IPAddress)
	assert.Equal(t, int64(1), page.Pagination.TotalCount)
}

func TestScanopy_LimitZeroMeansNoLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "0", r.URL.Query().Get("limit"))
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[[]Host]{
			Success: true,
			Data:    []Host{},
			Meta:    Meta{Pagination: &PaginationMeta{Limit: 0, HasMore: false}},
		})
	}))
	defer server.Close()
	_, err := NewScanopy(server.URL, "tok").ListHosts(context.Background(), ListParams{Limit: LimitOf(0)})
	require.NoError(t, err)
}

func TestScanopy_GetHostAndLists(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/hosts/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hosts/h1", r.URL.Path)
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[Host]{
			Success: true,
			Data:    Host{ID: "h1", Name: "web", NetworkID: "net-1"},
		})
	})
	mux.HandleFunc("/api/v1/networks", func(w http.ResponseWriter, _ *http.Request) {
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[[]Network]{
			Success: true,
			Data:    []Network{{ID: "net-1", Name: "Home"}},
			Meta:    Meta{Pagination: &PaginationMeta{TotalCount: 1}},
		})
	})
	mux.HandleFunc("/api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "h1", r.URL.Query().Get("host_id"))
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[[]Service]{
			Success: true,
			Data:    []Service{{ID: "s1", Name: "nginx", HostID: "h1"}},
		})
	})
	mux.HandleFunc("/api/v1/daemons", func(w http.ResponseWriter, _ *http.Request) {
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[[]Daemon]{
			Success: true,
			Data:    []Daemon{{ID: "d1", Name: "daemon-1", Version: "0.17.7"}},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewScanopy(server.URL, "tok")

	host, err := client.GetHost(context.Background(), "h1")
	require.NoError(t, err)
	assert.Equal(t, "web", host.Name)

	nets, err := client.ListNetworks(context.Background(), ListParams{})
	require.NoError(t, err)
	assert.Equal(t, "Home", nets.Items[0].Name)

	svcs, err := client.ListServices(context.Background(), ListParams{HostID: "h1"})
	require.NoError(t, err)
	assert.Equal(t, "nginx", svcs.Items[0].Name)

	daemons, err := client.ListDaemons(context.Background(), ListParams{})
	require.NoError(t, err)
	assert.Equal(t, "daemon-1", daemons.Items[0].Name)

	_, err = client.GetHost(context.Background(), "")
	assert.Error(t, err)
}

func TestScanopy_ListErrorEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(Envelope[any]{Success: false, Error: "forbidden"})
	}))
	defer server.Close()
	_, err := NewScanopy(server.URL, "tok").ListHosts(context.Background(), ListParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}
