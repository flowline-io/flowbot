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

// unreachableAddr is a loopback address on a different IP than httptest.NewServer's 127.0.0.1.
// Using a separate IP prevents the kernel from reassigning a closed server's port to another
// parallel test server, which would cause the subsequent request to land on the wrong handler.
const unreachableAddr = "http://127.0.0.2:1"

func TestGetClient_Defaults(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantURL string
	}{
		{
			name:    "empty config defaults to jsonplaceholder",
			configs: json.RawMessage(`{}`),
			wantURL: "https://jsonplaceholder.typicode.com",
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
			wantURL:  "https://jsonplaceholder.typicode.com",
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
		name    string
		handler http.HandlerFunc
		path    string
		wantErr bool
	}{
		{
			name: "successful get returns post",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				assert.Equal(t, "/posts/42", r.URL.Path)
				_, _ = w.Write([]byte(`{"userId":1,"id":42,"title":"hello","body":"world"}`))
			},
			path: "42",
		},
		{
			name: "empty path defaults to post 1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/posts/1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":1,"title":"default","body":"post"}`))
			},
		},
		{
			name:    "connection error returns err",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
			resp, err := c.Get(context.Background(), tt.path)
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
		name    string
		data    any
		wantErr bool
	}{
		{
			name: "successful post returns response",
			data: map[string]string{"title": "test", "body": "bar", "userId": "1"},
		},
		{
			name: "post with nil data",
			data: nil,
		},
		{
			name:    "connection error",
			data:    map[string]string{"title": "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var callCount atomic.Int32
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					callCount.Add(1)
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/posts", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"userId":1,"id":101,"title":"test","body":"bar"}`))
				}))
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
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
		name    string
		path    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful put",
			path: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/posts/1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":1,"title":"updated","body":"new body"}`))
			},
		},
		{
			name:    "connection error",
			wantErr: true,
		},
		{
			name: "put with data and path",
			path: "99",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/posts/99", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":99,"title":"patched","body":"data"}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
			resp, err := c.Put(context.Background(), tt.path, map[string]string{"key": "val"})
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
		name    string
		path    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "successful delete",
			path: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/posts/1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
		{
			name:    "connection error",
			wantErr: true,
		},
		{
			name: "delete with path",
			path: "42",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/posts/42", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
			resp, err := c.Delete(context.Background(), tt.path)
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
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "status 200 for existing post",
			code: 1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":1,"title":"hello","body":"world"}`))
			},
			wantErr: false,
		},
		{
			name: "post that exists returns ok",
			code: 42,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":42,"title":"hello","body":"world"}`))
			},
			wantErr: false,
		},
		{
			name:    "connection error returns err",
			code:    1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
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
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/posts/1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"userId":1,"id":1,"title":"hello","body":"world"}`))
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
		name         string
		cursor       string
		wantPage     string
		wantNext     string
		wantItemLen  int
		wantErr      bool
	}{
		{name: "first page without cursor", cursor: "", wantPage: "1", wantNext: "2", wantItemLen: 1},
		{name: "explicit page one", cursor: "1", wantPage: "1", wantNext: "2", wantItemLen: 1},
		{name: "second page", cursor: "2", wantPage: "2", wantNext: "3", wantItemLen: 1},
		{name: "connection error", cursor: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var endpoint string
			if tt.wantErr {
				endpoint = unreachableAddr
			} else {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/posts", r.URL.Path)
					assert.Equal(t, tt.wantPage, r.URL.Query().Get("_page"))
					assert.Equal(t, "1", r.URL.Query().Get("_limit"))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[{"userId":1,"id":1,"title":"hello","body":"world"}]`))
				}))
				defer srv.Close()
				endpoint = srv.URL
			}
			c := NewExample(endpoint, "")
			items, next, err := c.ListRawEvents(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, tt.wantItemLen)
			assert.Equal(t, tt.wantNext, next)
		})
	}
}

func TestOAuthMethods(t *testing.T) {
	t.Parallel()
	t.Run("GetAuthorizeURL returns URL", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://jsonplaceholder.typicode.com", "")
		url := c.GetAuthorizeURL("test-state")
		assert.Contains(t, url, "https://jsonplaceholder.typicode.com/authorize")
		assert.Contains(t, url, "state=test-state")
	})
	t.Run("GetAccessToken returns OAuthToken", func(t *testing.T) {
		t.Parallel()
		c := NewExample("https://jsonplaceholder.typicode.com", "")
		tk, err := c.GetAccessToken(nil)
		require.NoError(t, err)
		assert.Equal(t, "example-token", tk.AccessToken)
	})
	t.Run("OAuthProvider interface compliance", func(_ *testing.T) {
		var _ providers.OAuthProvider = NewExample("https://jsonplaceholder.typicode.com", "")
	})
}

func TestListRawEvents_ContextCanceled(t *testing.T) {
	t.Run("canceled context returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"userId":1,"id":1,"title":"hello","body":"world"}]`))
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		c := NewExample(srv.URL, "")
		_, _, err := c.ListRawEvents(ctx, "")
		assert.Error(t, err)
	})
}
