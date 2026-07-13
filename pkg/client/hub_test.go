package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubListApps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "lists apps",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":[
						{"name":"app1","path":"/opt/app1","status":"running","health":"healthy"},
						{"name":"app2","path":"/opt/app2","status":"stopped","health":"unknown"}
					]
				}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty app list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"hub unavailable"}`))
			},
			wantErr:    true,
			errContain: "hub unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.ListApps(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestHubGetApp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantName   string
		wantErr    bool
		errContain string
	}{
		{
			name: "app found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"myapp","path":"/opt/myapp","status":"running","health":"healthy"}}`))
			},
			wantName: "myapp",
			wantErr:  false,
		},
		{
			name: "app not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"app not found"}`))
			},
			wantErr:    true,
			errContain: "app not found",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"internal error"}`))
			},
			wantErr:    true,
			errContain: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.GetApp(context.Background(), "myapp")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantName, result.Name)
		})
	}
}

func TestHubGetAppStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus string
		wantErr    bool
	}{
		{
			name: "app status running",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"myapp","status":"running"}}`))
			},
			wantStatus: "running",
			wantErr:    false,
		},
		{
			name: "app status stopped",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"myapp","status":"stopped"}}`))
			},
			wantStatus: "stopped",
			wantErr:    false,
		},
		{
			name: "app not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.GetAppStatus(context.Background(), "myapp")

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestHubGetAppLogs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		tail       int
		handler    http.HandlerFunc
		wantLogs   int
		wantErr    bool
		errContain string
	}{
		{
			name: "logs with tail",
			tail: 10,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"myapp","status":"running","logs":["log1","log2","log3"]}}`))
			},
			wantLogs: 3,
			wantErr:  false,
		},
		{
			name: "logs without tail",
			tail: 0,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"myapp","status":"running","logs":["log1"]}}`))
			},
			wantLogs: 1,
			wantErr:  false,
		},
		{
			name: "app error",
			tail: 0,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"app not found"}`))
			},
			wantErr:    true,
			errContain: "app not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.GetAppLogs(context.Background(), "myapp", tt.tail)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Logs, tt.wantLogs)
		})
	}
}

func TestHubListCapabilities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "lists capabilities",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":[
						{"type":"karakeep","app":"karakeep","healthy":true},
						{"type":"kanboard","app":"kanboard","healthy":true}
					]
				}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty capabilities",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.ListCapabilities(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestHubGetCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantType   string
		wantErr    bool
		errContain string
	}{
		{
			name: "capability found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"type":"karakeep","app":"karakeep","healthy":true}}`))
			},
			wantType: "karakeep",
			wantErr:  false,
		},
		{
			name: "capability not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"capability not found"}`))
			},
			wantErr:    true,
			errContain: "capability not found",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.GetCapability(context.Background(), "karakeep")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantType, result.Type)
		})
	}
}

func TestHubGetHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus string
		wantErr    bool
		errContain string
	}{
		{
			name: "healthy",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"status":"healthy","timestamp":"2025-01-01T00:00:00Z","details":[],"app_statuses":[]}}`))
			},
			wantStatus: "healthy",
			wantErr:    false,
		},
		{
			name: "degraded",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"status":"degraded","timestamp":"2025-01-01T00:00:00Z"}}`))
			},
			wantStatus: "degraded",
			wantErr:    false,
		},
		{
			name: "error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"failed","message":"health check failed"}`))
			},
			wantErr:    true,
			errContain: "health check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Hub.GetHealth(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestHubStartStopRestartApp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "start app success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"started":true}}`))
			},
			wantErr: false,
		},
		{
			name: "start app error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"app not found"}`))
			},
			wantErr:    true,
			errContain: "app not found",
		},
		{
			name: "stop app success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"stopped":true}}`))
			},
			wantErr: false,
		},
		{
			name: "restart app success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"restarted":true}}`))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			ctx := context.Background()

			var err error
			switch {
			case tt.name == "start app success" || tt.name == "start app error":
				_, err = c.Hub.StartApp(ctx, "myapp")
			case tt.name == "stop app success":
				_, err = c.Hub.StopApp(ctx, "myapp")
			case tt.name == "restart app success":
				_, err = c.Hub.RestartApp(ctx, "myapp")
			}

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

func TestIntToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int
		want  string
	}{
		{
			name:  "zero",
			input: 0,
			want:  "0",
		},
		{
			name:  "positive single digit",
			input: 5,
			want:  "5",
		},
		{
			name:  "positive multi-digit",
			input: 12345,
			want:  "12345",
		},
		{
			name:  "negative number",
			input: -42,
			want:  "-42",
		},
		{
			name:  "negative large",
			input: -1000,
			want:  "-1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := intToString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
