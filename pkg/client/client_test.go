package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"resty.dev/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		serverURL string
		token     string
	}{
		{
			name:      "valid url and token",
			serverURL: "http://localhost:6060",
			token:     "token-abc123",
		},
		{
			name:      "empty token",
			serverURL: "http://localhost:6060",
			token:     "",
		},
		{
			name:      "trailing slash url",
			serverURL: "http://localhost:6060/",
			token:     "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewClient(tt.serverURL, tt.token)

			require.NotNil(t, c)
			assert.Equal(t, tt.serverURL, c.baseURL)
			assert.NotNil(t, c.rc)
			assert.NotNil(t, c.RawRequest())
			assert.NotNil(t, c.Kanban)
			assert.NotNil(t, c.Bookmark)
			assert.NotNil(t, c.Reader)
			assert.NotNil(t, c.User)
			assert.NotNil(t, c.Search)
			assert.NotNil(t, c.Dev)
			assert.NotNil(t, c.Server)
			assert.NotNil(t, c.Hub)
			assert.NotNil(t, c.Pipeline)
			assert.NotNil(t, c.Workflow)
			assert.NotNil(t, c.Forge)
			assert.NotNil(t, c.Github)
			assert.NotNil(t, c.Memo)
			assert.NotNil(t, c.Trilium)
			assert.NotNil(t, c.Fireflyiii)
			assert.NotNil(t, c.Transmission)
			assert.NotNil(t, c.Nocodb)
		})
	}
}

func TestSetTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "positive timeout",
			timeout: 60 * time.Second,
		},
		{
			name:    "zero timeout",
			timeout: 0,
		},
		{
			name:    "short timeout",
			timeout: 1 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewClient("http://localhost", "token")
			c.SetTimeout(tt.timeout)
			assert.NotNil(t, c.rc)
		})
	}
}

func TestSetDebug(t *testing.T) {
	t.Parallel()

	t.Run("enable debug sets flag", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		assert.False(t, c.debugErrorHookSet)
		assert.False(t, c.DebugEnabled())

		c.SetDebug(true)
		assert.True(t, c.debugErrorHookSet)
		assert.True(t, c.DebugEnabled())
	})

	t.Run("disable debug does not toggle flag", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")

		c.SetDebug(false)
		assert.False(t, c.debugErrorHookSet)
		assert.False(t, c.DebugEnabled())
	})

	t.Run("double enable only sets once", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")

		c.SetDebug(true)
		assert.True(t, c.debugErrorHookSet)

		c.debugErrorHookSet = false
		c.SetDebug(true)
		assert.True(t, c.debugErrorHookSet)
	})
}

func TestClientRawRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil request", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		req := c.RawRequest()
		assert.NotNil(t, req)
	})

	t.Run("multiple calls return fresh requests", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		req1 := c.RawRequest()
		req2 := c.RawRequest()
		assert.NotNil(t, req1)
		assert.NotNil(t, req2)
	})

	t.Run("request is associated with client", func(t *testing.T) {
		t.Parallel()
		c := NewClient("http://localhost", "token")
		req := c.RawRequest()
		assert.NotNil(t, req)
		assert.NotNil(t, req.Header)
	})
}

func TestHttpGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "successful response with data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"test"}}`))
			},
			wantErr: false,
		},
		{
			name: "api error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10001","message":"bad request"}`))
			},
			wantErr:    true,
			errContain: "bad request",
		},
		{
			name: "empty response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:    true,
			errContain: "empty response",
		},
		{
			name: "invalid json response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`not json`))
			},
			wantErr:    true,
			errContain: "parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			var result map[string]any
			err := c.Get(context.Background(), "/test", &result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "test", result["name"])
		})
	}
}

func TestHttpPost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		body       any
		wantErr    bool
		errContain string
	}{
		{
			name: "successful post with data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1}}`))
			},
			body:    map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name: "post with api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"server error"}`))
			},
			body:       nil,
			wantErr:    true,
			errContain: "server error",
		},
		{
			name: "post with empty response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:    true,
			errContain: "empty response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			var result map[string]any
			err := c.Post(context.Background(), "/test", tt.body, &result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHttpPatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "successful patch",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
			wantErr: false,
		},
		{
			name: "patch with not found error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
			},
			wantErr:    true,
			errContain: "not found",
		},
		{
			name: "patch with empty body response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:    true,
			errContain: "empty response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			var result map[string]any
			err := c.Patch(context.Background(), "/test", map[string]string{"a": "b"}, &result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHttpPut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "successful put",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
			wantErr: false,
		},
		{
			name: "put with api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid payload"}`))
			},
			wantErr:    true,
			errContain: "invalid payload",
		},
		{
			name: "put with jSON parse error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`invalid json {{{`))
			},
			wantErr:    true,
			errContain: "parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			var result map[string]any
			err := c.Put(context.Background(), "/test", map[string]string{"x": "y"}, &result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHttpDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "successful delete",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"deleted":true}}`))
			},
			wantErr: false,
		},
		{
			name: "delete not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"resource not found"}`))
			},
			wantErr:    true,
			errContain: "resource not found",
		},
		{
			name: "delete empty response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:    true,
			errContain: "empty response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			var result map[string]any
			err := c.Delete(context.Background(), "/test", nil, &result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParseResponse(t *testing.T) {
	t.Parallel()

	t.Run("success with data", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"ok","data":{"key":"value","num":42}}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		var result map[string]any
		err := parseResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
		assert.InEpsilon(t, float64(42), result["num"], 0)
	})

	t.Run("success with nil data field", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"ok"}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.NoError(t, err)
	})

	t.Run("success with data array", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"ok","data":[1,2,3]}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		var result []int
		err := parseResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Contains(t, apiErr.Message, "empty response")
	})

	t.Run("invalid json body", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`not valid json {{{`))
		}))
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse response")
	})

	t.Run("api failed with retcode and message", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"failed","retcode":"10009","message":"resource not found"}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, "10009", apiErr.RetCode)
		assert.Equal(t, "resource not found", apiErr.Message)
	})

	t.Run("api failed without retcode", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"failed","message":"something went wrong"}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Empty(t, apiErr.RetCode)
		assert.Equal(t, "something went wrong", apiErr.Message)
	})

	t.Run("api failed with empty message defaults to unknown error", func(t *testing.T) {
		t.Parallel()
		server := newJSONServer(`{"status":"failed"}`)
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, "unknown error", apiErr.Message)
	})
}

func TestParseResponse_StatusCode(t *testing.T) {
	t.Parallel()

	t.Run("preserves http status code 404", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
		}))
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	})

	t.Run("preserves http status code 500", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"failed","message":"boom"}`))
		}))
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	})

	t.Run("preserves http status code 200 for json errors", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{{{ broken`))
		}))
		defer server.Close()

		resp := mustGetRestyResp(t, server.URL, "/test")
		err := parseResponse(resp, nil)
		require.Error(t, err)
		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusOK, apiErr.StatusCode)
	})
}

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		retCode    string
		message    string
		want       string
	}{
		{
			name:       "with retcode",
			statusCode: 404,
			retCode:    "10009",
			message:    "not found",
			want:       "API error (code 10009): not found",
		},
		{
			name:       "without retcode uses status code",
			statusCode: 500,
			retCode:    "",
			message:    "internal error",
			want:       "API error (status 500): internal error",
		},
		{
			name:       "empty message with retcode",
			statusCode: 400,
			retCode:    "10001",
			message:    "",
			want:       "API error (code 10001): ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &APIError{
				StatusCode: tt.statusCode,
				RetCode:    tt.retCode,
				Message:    tt.message,
			}
			assert.Equal(t, tt.want, e.Error())
		})
	}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "api error with status 404",
			err:  &APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			want: true,
		},
		{
			name: "api error with retcode 10009",
			err:  &APIError{StatusCode: http.StatusBadRequest, RetCode: "10009", Message: "not found"},
			want: true,
		},
		{
			name: "api error with different status and retcode",
			err:  &APIError{StatusCode: http.StatusInternalServerError, RetCode: "10000", Message: "error"},
			want: false,
		},
		{
			name: "non api error",
			err:  fmt.Errorf("network error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsNotFound(tt.err))
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "api error with status 401",
			err:  &APIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"},
			want: true,
		},
		{
			name: "api error with retcode 60005",
			err:  &APIError{StatusCode: http.StatusForbidden, RetCode: "60005", Message: "unauthorized"},
			want: true,
		},
		{
			name: "api error with different status",
			err:  &APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			want: false,
		},
		{
			name: "non api error",
			err:  fmt.Errorf("random error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsUnauthorized(tt.err))
		})
	}
}

func TestStringOr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		kv         map[string]any
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "key exists with string value",
			kv:         map[string]any{"title": "hello"},
			key:        "title",
			defaultVal: "default",
			want:       "hello",
		},
		{
			name:       "key exists with non-string value",
			kv:         map[string]any{"count": 42},
			key:        "count",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "key missing returns default",
			kv:         map[string]any{"other": "value"},
			key:        "missing",
			defaultVal: "fallback",
			want:       "fallback",
		},
		{
			name:       "empty map returns default",
			kv:         map[string]any{},
			key:        "any",
			defaultVal: "none",
			want:       "none",
		},
		{
			name:       "nil map returns default",
			kv:         nil,
			key:        "key",
			defaultVal: "safe",
			want:       "safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stringOr(tt.kv, tt.key, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// newJSONServer creates an httptest server returning the given JSON body with status 200.
func newJSONServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

// mustGetRestyResp makes a real resty GET request and returns the response.
func mustGetRestyResp(t *testing.T, serverURL, path string) *resty.Response {
	t.Helper()
	rc := resty.New()
	resp, err := rc.R().Get(serverURL + path)
	require.NoError(t, err)
	return resp
}
