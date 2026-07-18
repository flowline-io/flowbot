package netalertx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetAlertX(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:20211", wantNil: false},
		{name: "with path", endpoint: "http://localhost:20211/", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewNetAlertX(tt.endpoint, "tok")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNetAlertX_ListDevices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       DevicesResponse
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: DevicesResponse{
				Success: true,
				Devices: []Device{{Name: "Router", MAC: "AA:BB:CC:DD:EE:FF", Status: "online"}},
			},
			wantLen: 1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       DevicesResponse{Success: true, Devices: []Device{}},
			wantLen:    0,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/devices", r.URL.Path)
				assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
				}
			}))
			defer server.Close()
			got, err := NewNetAlertX(server.URL, "tok").ListDevices(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestNetAlertX_TotalsSearchTopology(t *testing.T) {
	t.Parallel()
	t.Run("totals success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/devices/totals", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode([]int{10, 8, 1, 0, 1, 0})
		}))
		defer server.Close()
		got, err := NewNetAlertX(server.URL, "tok").GetTotals(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 10, got.All)
		assert.Equal(t, 8, got.Connected)
		assert.Equal(t, 1, got.Down)
	})
	t.Run("search success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/devices/search", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode(SearchResponse{
				Success: true,
				Devices: []Device{{Name: "PC", LastIP: "192.168.1.50"}},
			})
		}))
		defer server.Close()
		got, err := NewNetAlertX(server.URL, "tok").SearchDevices(context.Background(), "192.168.1")
		require.NoError(t, err)
		assert.Equal(t, "PC", got[0].Name)
	})
	t.Run("search requires query", func(t *testing.T) {
		t.Parallel()
		_, err := NewNetAlertX("http://localhost", "tok").SearchDevices(context.Background(), "")
		assert.Error(t, err)
	})
	t.Run("health success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/devices/totals", r.URL.Path)
			_ = sonic.ConfigDefault.NewEncoder(w).Encode([]int{1, 1, 0, 0, 0, 0})
		}))
		defer server.Close()
		assert.NoError(t, NewNetAlertX(server.URL, "tok").Health(context.Background()))
	})
}
