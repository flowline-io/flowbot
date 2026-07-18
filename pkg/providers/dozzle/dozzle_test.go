package dozzle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDozzle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		wantNil  bool
	}{
		{name: "empty endpoint", endpoint: "", wantNil: true},
		{name: "with endpoint", endpoint: "http://localhost:8080", wantNil: false},
		{name: "with path", endpoint: "http://localhost:8080/dozzle", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewDozzle(tt.endpoint, "", "", "")
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestDozzle_Health(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "healthy", statusCode: http.StatusOK},
		{name: "unhealthy", statusCode: http.StatusInternalServerError, wantErr: true},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/healthcheck", r.URL.Path)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()
			err := NewDozzle(server.URL, "", "", "").Health(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDozzle_Version(t *testing.T) {
	t.Parallel()
	t.Run("success with bearer", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/version", r.URL.Path)
			assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("v8.0.0\n"))
		}))
		defer server.Close()
		got, err := NewDozzle(server.URL, "", "", "jwt").Version(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "v8.0.0", got.Version)
	})
	t.Run("error status", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()
		_, err := NewDozzle(server.URL, "u", "p", "").Version(context.Background())
		assert.Error(t, err)
	})
}
