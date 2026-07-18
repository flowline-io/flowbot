package beszel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBeszel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint returns nil", endpoint: "", wantNil: true},
		{name: "configured endpoint", endpoint: "http://localhost:8090", wantNil: false},
		{name: "with path", endpoint: "http://localhost:8090/", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewBeszel(tt.endpoint, "tok", "", "")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestBeszel_ListSystems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       SystemList
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			body: SystemList{
				Page: 1, PerPage: 30, TotalItems: 1,
				Items: []System{{ID: "sys1", Name: "host-a", Status: "up"}},
			},
			wantCount: 1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       SystemList{Items: []System{}},
			wantCount:  0,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/collections/systems/records", r.URL.Path)
				assert.Equal(t, "test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
				}
			}))
			defer server.Close()

			client := NewBeszel(server.URL, "test-token", "", "")
			result, err := client.ListSystems(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantCount)
		})
	}
}

func TestBeszel_GetSystem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         string
		statusCode int
		body       System
		wantErr    bool
	}{
		{
			name:       "successful get",
			id:         "sys1",
			statusCode: http.StatusOK,
			body:       System{ID: "sys1", Name: "host-a", Status: "up"},
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
		{
			name:       "not found",
			id:         "missing",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.id == "" {
				client := NewBeszel("http://localhost", "tok", "", "")
				_, err := client.GetSystem(context.Background(), tt.id)
				assert.Error(t, err)
				return
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/collections/systems/records/"+tt.id, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
				}
			}))
			defer server.Close()

			client := NewBeszel(server.URL, "tok", "", "")
			result, err := client.GetSystem(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.body.ID, result.ID)
			assert.Equal(t, tt.body.Name, result.Name)
		})
	}
}

func TestBeszel_AuthWithPassword(t *testing.T) {
	t.Parallel()
	t.Run("password auth then list", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/collections/users/auth-with-password":
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(AuthResponse{Token: "jwt-token"})
			case "/api/collections/systems/records":
				assert.Equal(t, "jwt-token", r.Header.Get("Authorization"))
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(SystemList{Items: []System{{ID: "1", Name: "a"}}})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := NewBeszel(server.URL, "", "user@example.com", "secret")
		result, err := client.ListSystems(context.Background())
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
	})
}
