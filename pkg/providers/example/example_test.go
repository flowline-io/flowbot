package example

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetClient_Defaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		configs json.RawMessage
		wantURL string
	}{
		{
			name:    "empty config defaults to httpbin",
			configs: json.RawMessage(`{}`),
			wantURL: "https://httpbin.org",
		},
		{
			name:    "custom endpoint",
			configs: json.RawMessage(`{"example":{"endpoint":"https://custom.example.com"}}`),
			wantURL: "https://custom.example.com",
		},
		{
			name:    "config with token",
			configs: json.RawMessage(`{"example":{"endpoint":"https://custom.example.com","token":"abc123"}}`),
			wantURL: "https://custom.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			providers.Configs = tt.configs
			c := GetClient()
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL())
		})
	}
}

func TestNewExample(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantURL  string
	}{
		{
			name:     "explicit endpoint",
			endpoint: "https://api.example.com",
			token:    "",
			wantURL:  "https://api.example.com",
		},
		{
			name:     "empty endpoint defaults",
			endpoint: "",
			token:    "",
			wantURL:  "https://httpbin.org",
		},
		{
			name:     "with auth token",
			endpoint: "https://api.example.com",
			token:    "test-token",
			wantURL:  "https://api.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewExample(tt.endpoint, tt.token)
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL())
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		wantErr  bool
		closeSrv bool
	}{
		{
			name: "successful get returns response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				assert.Equal(t, "/get", r.URL.Path)
				_, _ = w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4","method":"GET"}`))
			},
		},
		{
			name: "get with path sets query param",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "my-path", r.URL.Query().Get("path"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/get","method":"GET"}`))
			},
		},
		{
			name:     "connection error returns err",
			wantErr:  true,
			closeSrv: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := tt.handler
			if handler == nil {
				handler = func(_ http.ResponseWriter, _ *http.Request) {}
			}
			srv := httptest.NewServer(handler)
			c := NewExample(srv.URL, "")
			if tt.closeSrv {
				srv.Close()
			} else {
				defer srv.Close()
			}
			var path string
			if tt.name == "get with path sets query param" {
				path = "my-path"
			}
			resp, err := c.Get(context.Background(), path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestPost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		closeSrv bool
	}{
		{
			name: "successful post returns response",
			data: map[string]string{"title": "test"},
		},
		{
			name: "post with nil data",
			data: nil,
		},
		{
			name:     "connection error",
			data:     map[string]string{"title": "test"},
			wantErr:  true,
			closeSrv: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var callCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount.Add(1)
				assert.Equal(t, "POST", r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/post","method":"POST"}`))
			}))
			c := NewExample(srv.URL, "")
			if tt.closeSrv {
				srv.Close()
			} else {
				defer srv.Close()
			}
			resp, err := c.Post(context.Background(), "", tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, int32(1), callCount.Load())
		})
	}
}

func TestPut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		wantErr  bool
		closeSrv bool
	}{
		{
			name: "successful put",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/put","method":"PUT"}`))
			},
		},
		{
			name:     "connection error",
			wantErr:  true,
			closeSrv: true,
		},
		{
			name: "put with data",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/put","method":"PUT"}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := tt.handler
			if handler == nil {
				handler = func(_ http.ResponseWriter, _ *http.Request) {}
			}
			srv := httptest.NewServer(handler)
			c := NewExample(srv.URL, "")
			if tt.closeSrv {
				srv.Close()
			} else {
				defer srv.Close()
			}
			resp, err := c.Put(context.Background(), "", map[string]string{"key": "val"})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		wantErr  bool
		closeSrv bool
	}{
		{
			name: "successful delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/delete","method":"DELETE"}`))
			},
		},
		{
			name:     "connection error",
			wantErr:  true,
			closeSrv: true,
		},
		{
			name: "delete with path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "my-resource", r.URL.Query().Get("path"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/delete","method":"DELETE"}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := tt.handler
			if handler == nil {
				handler = func(_ http.ResponseWriter, _ *http.Request) {}
			}
			srv := httptest.NewServer(handler)
			c := NewExample(srv.URL, "")
			if tt.closeSrv {
				srv.Close()
			} else {
				defer srv.Close()
			}
			var path string
			if tt.name == "delete with path" {
				path = "my-resource"
			}
			resp, err := c.Delete(context.Background(), path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		code    int
		wantErr bool
	}{
		{name: "status 200", code: 200, wantErr: false},
		{name: "status 418", code: 418, wantErr: false},
		{name: "status 404", code: 404, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.code)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/status","method":"GET"}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			resp, err := c.GetStatus(context.Background(), tt.code)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetWithDelay(t *testing.T) {
	tests := []struct {
		name      string
		seconds   int
		wantErr   bool
		cancelCtx bool
	}{
		{name: "short delay succeeds", seconds: 0, wantErr: false},
		{name: "short delay works", seconds: 1, wantErr: false},
		{name: "context expiry", seconds: 30, wantErr: true, cancelCtx: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/delay","method":"GET"}`))
			}))
			defer srv.Close()
			c := NewExample(srv.URL, "")
			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			resp, err := c.GetWithDelay(ctx, tt.seconds)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cursor   string
		wantErr  bool
		closeSrv bool
	}{
		{name: "list with no cursor", cursor: ""},
		{name: "list with cursor", cursor: "next-page"},
		{name: "connection error", cursor: "", wantErr: true, closeSrv: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.cursor != "" {
					assert.Equal(t, tt.cursor, r.URL.Query().Get("cursor"))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4","headers":{}}`))
			}))
			c := NewExample(srv.URL, "")
			if tt.closeSrv {
				srv.Close()
			} else {
				defer srv.Close()
			}
			items, next, err := c.ListRawEvents(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, items)
			assert.Empty(t, next)
		})
	}
}

func TestOAuthMethods(t *testing.T) {
	t.Parallel()
	t.Run("GetAuthorizeURL returns URL", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://httpbin.org", "")
		url := c.GetAuthorizeURL()
		assert.Contains(t, url, "https://httpbin.org/authorize")
	})
	t.Run("GetAccessToken returns token KV", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://httpbin.org", "")
		kv, err := c.GetAccessToken(nil)
		require.NoError(t, err)
		assert.Equal(t, "example-token", kv["access_token"])
	})
	t.Run("OAuthProvider interface compliance", func(t *testing.T) {
		t.Parallel()
		var _ providers.OAuthProvider = NewExample("https://httpbin.org", "")
	})
}

func TestListRawEvents_ContextCanceled(t *testing.T) {
	t.Run("canceled context returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"https://example.com/get","origin":"1.2.3.4"}`))
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c := NewExample(srv.URL, "")
		_, _, err := c.ListRawEvents(ctx, "")
		assert.Error(t, err)
	})
}
