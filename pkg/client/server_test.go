package client

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestServerStacktrace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "stacktrace success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":{
						"timestamp":"2025-01-01T00:00:00Z",
						"stack_trace":"goroutine 1 [running]:\nmain.main()\n\tmain.go:10",
						"process":{"name":"server","pid":12345,"create_time":"2025-01-01","uptime_seconds":3600,"cpu_percent":5.5,"num_threads":10,"memory":{"rss_bytes":50000000,"vms_bytes":100000000,"swap_bytes":0,"rss_mb":47.68,"vms_mb":95.37}},
						"runtime":{"go_version":"1.26","num_cpu":8,"num_goroutine":42,"memory":{"alloc_bytes":1000000,"total_alloc_bytes":5000000,"sys_bytes":20000000,"heap_alloc_bytes":800000,"heap_sys_bytes":3000000,"alloc_mb":0.95,"sys_mb":19.07},"gc":{"num_gc":100,"pause_total_ns":50000000,"last_gc":"2025-01-01T00:05:00Z"}},
						"build_info":{"go_version":"1.26","main_module":"flowbot","main_version":"v1.0.0","settings":{"CGO_ENABLED":"1"}}
					}
				}`))
			},
		},
		{
			name: "stacktrace minimal data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"timestamp":"now"}}`))
			},
		},
		{
			name: "missing process and runtime",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"timestamp":"now","stack_trace":"trace here"}}`))
			},
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"diagnostics unavailable"}`))
			},
			wantErr:    true,
			errContain: "diagnostics unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Server.Stacktrace(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.name == "stacktrace success" {
				assert.Equal(t, "server", result.Process.Name)
				assert.Equal(t, 12345, result.Process.PID)
				assert.InEpsilon(t, 3600.0, result.Process.UptimeSeconds, 0)
				assert.InEpsilon(t, 5.5, result.Process.CPUPercent, 0)
				assert.Equal(t, int32(10), result.Process.NumThreads)
				assert.Equal(t, "1.26", result.Runtime.GoVersion)
				assert.Equal(t, 8, result.Runtime.NumCPU)
				assert.Equal(t, 42, result.Runtime.NumGoroutine)
				assert.Equal(t, uint32(100), result.Runtime.GC.NumGC)
				assert.Equal(t, "v1.0.0", result.BuildInfo.MainVersion)
			}
		})
	}
}

func TestServerUploadMultipart(t *testing.T) {
	t.Parallel()

	t.Run("upload multipart success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true,"result":["http://cdn/f.png"]}}`))
		}))
		defer server.Close()

		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		part, err := writer.CreateFormFile("file", "test.png")
		require.NoError(t, err)
		_, err = part.Write([]byte("image data"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		c := NewClient(server.URL, "token")
		result, err := c.Server.UploadMultipart(context.Background(), writer, bytes.NewReader(buf.Bytes()))
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("upload multipart api error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":"failed","message":"invalid form data"}`))
		}))
		defer server.Close()

		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		err := writer.Close()
		require.NoError(t, err)

		c := NewClient(server.URL, "token")
		_, err = c.Server.UploadMultipart(context.Background(), writer, bytes.NewReader(buf.Bytes()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid form data")
	})

	t.Run("upload multipart parse error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`not json`))
		}))
		defer server.Close()

		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		require.NoError(t, writer.Close())

		c := NewClient(server.URL, "token")
		_, err := c.Server.UploadMultipart(context.Background(), writer, bytes.NewReader(buf.Bytes()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse response")
	})
}

func TestValidateUploadFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      map[string]io.Reader
		filenames  map[string]string
		wantErr    bool
		errContain string
	}{
		{
			name:      "valid single file",
			files:     map[string]io.Reader{"file1": bytes.NewReader([]byte("content"))},
			filenames: map[string]string{"file1": "file.txt"},
			wantErr:   false,
		},
		{
			name:       "empty files",
			files:      map[string]io.Reader{},
			filenames:  map[string]string{},
			wantErr:    true,
			errContain: "at least one file is required",
		},
		{
			name:       "empty field name",
			files:      map[string]io.Reader{"": bytes.NewReader([]byte("content"))},
			filenames:  map[string]string{},
			wantErr:    true,
			errContain: "field name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateUploadFiles(tt.files, tt.filenames)
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

func TestConvertStacktraceResult(t *testing.T) {
	t.Parallel()

	t.Run("full data", func(t *testing.T) {
		t.Parallel()
		data := types.KV{
			"timestamp":   "2025-01-01T00:00:00Z",
			"stack_trace": "goroutine trace...",
			"process": map[string]any{
				"name":           "server",
				"pid":            float64(12345),
				"create_time":    "2025-01-01",
				"uptime_seconds": float64(3600),
				"cpu_percent":    float64(5.5),
				"num_threads":    float64(10),
				"memory": map[string]any{
					"rss_bytes":  float64(50000),
					"vms_bytes":  float64(100000),
					"swap_bytes": float64(0),
					"rss_mb":     float64(47.6),
					"vms_mb":     float64(95.3),
				},
			},
			"runtime": map[string]any{
				"go_version":    "1.26",
				"num_cpu":       float64(4),
				"num_goroutine": float64(50),
				"memory": map[string]any{
					"alloc_bytes":       float64(1000000),
					"total_alloc_bytes": float64(5000000),
					"sys_bytes":         float64(20000000),
					"heap_alloc_bytes":  float64(800000),
					"heap_sys_bytes":    float64(3000000),
					"alloc_mb":          float64(0.95),
					"sys_mb":            float64(19.07),
				},
				"gc": map[string]any{
					"num_gc":         float64(150),
					"pause_total_ns": float64(60000000),
					"last_gc":        "2025-01-01T00:05:00Z",
				},
			},
			"build_info": map[string]any{
				"go_version":   "1.26",
				"main_module":  "flowbot",
				"main_version": "v2.0.0",
				"settings": map[string]any{
					"CGO_ENABLED": "1",
				},
			},
		}

		result := convertStacktraceResult(data)
		require.NotNil(t, result)
		assert.Equal(t, "2025-01-01T00:00:00Z", result.Timestamp)
		assert.Equal(t, "goroutine trace...", result.StackTrace)
		assert.Equal(t, "server", result.Process.Name)
		assert.Equal(t, 12345, result.Process.PID)
		assert.InEpsilon(t, 3600.0, result.Process.UptimeSeconds, 0)
		assert.InEpsilon(t, 5.5, result.Process.CPUPercent, 0)
		assert.Equal(t, int32(10), result.Process.NumThreads)
		assert.Equal(t, int64(50000), result.Process.Memory.RSSBytes)
		assert.InEpsilon(t, 47.6, result.Process.Memory.RSSMB, 0)
		assert.Equal(t, "1.26", result.Runtime.GoVersion)
		assert.Equal(t, 4, result.Runtime.NumCPU)
		assert.Equal(t, 50, result.Runtime.NumGoroutine)
		assert.Equal(t, uint64(1000000), result.Runtime.Memory.AllocBytes)
		assert.InEpsilon(t, 19.07, result.Runtime.Memory.SysMB, 0)
		assert.Equal(t, uint32(150), result.Runtime.GC.NumGC)
		assert.Equal(t, uint64(60000000), result.Runtime.GC.PauseTotalNs)
		assert.Equal(t, "v2.0.0", result.BuildInfo.MainVersion)
		assert.Equal(t, "1", result.BuildInfo.Settings["CGO_ENABLED"])
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()
		result := convertStacktraceResult(types.KV{})
		require.NotNil(t, result)
		assert.Empty(t, result.Timestamp)
		assert.Empty(t, result.StackTrace)
		assert.Equal(t, Process{}, result.Process)
		assert.Equal(t, Runtime{}, result.Runtime)
	})

	t.Run("partial process data", func(t *testing.T) {
		t.Parallel()
		data := types.KV{
			"process": map[string]any{
				"name":        "worker",
				"pid":         float64(9999),
				"create_time": "2025-01-01",
			},
		}
		result := convertStacktraceResult(data)
		assert.Equal(t, "worker", result.Process.Name)
		assert.Equal(t, 9999, result.Process.PID)
		assert.Equal(t, "2025-01-01", result.Process.CreateTime)
		assert.InDelta(t, 0.0, result.Process.UptimeSeconds, 0)
	})
}

func TestConvertProcess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		want Process
	}{
		{
			name: "full process data",
			data: map[string]any{
				"name":           "server",
				"pid":            float64(100),
				"create_time":    "2025-01-01",
				"uptime_seconds": float64(7200),
				"cpu_percent":    float64(10.5),
				"num_threads":    float64(12),
				"memory": map[string]any{
					"rss_bytes": float64(1024),
					"vms_bytes": float64(2048),
				},
			},
			want: Process{
				Name:          "server",
				PID:           100,
				CreateTime:    "2025-01-01",
				UptimeSeconds: 7200,
				CPUPercent:    10.5,
				NumThreads:    12,
				Memory:        Memory{RSSBytes: 1024, VMSBytes: 2048},
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
			want: Process{},
		},
		{
			name: "string name only",
			data: map[string]any{"name": "test-process"},
			want: Process{Name: "test-process"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertProcess(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertMemory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		want Memory
	}{
		{
			name: "full memory data",
			data: map[string]any{
				"rss_bytes":  float64(50000),
				"vms_bytes":  float64(100000),
				"swap_bytes": float64(0),
				"rss_mb":     float64(47.6),
				"vms_mb":     float64(95.3),
			},
			want: Memory{
				RSSBytes:  50000,
				VMSBytes:  100000,
				SwapBytes: 0,
				RSSMB:     47.6,
				VMSMB:     95.3,
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
			want: Memory{},
		},
		{
			name: "only rss",
			data: map[string]any{"rss_bytes": float64(1234)},
			want: Memory{RSSBytes: 1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertMemory(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertRuntime(t *testing.T) {
	t.Parallel()

	t.Run("full runtime data", func(t *testing.T) {
		t.Parallel()
		data := map[string]any{
			"go_version":    "1.26",
			"num_cpu":       float64(8),
			"num_goroutine": float64(100),
			"memory": map[string]any{
				"alloc_bytes": float64(5000),
				"sys_bytes":   float64(10000),
			},
			"gc": map[string]any{
				"num_gc":         float64(50),
				"pause_total_ns": float64(30000),
				"last_gc":        "2025-01-01",
			},
		}
		got := convertRuntime(data)
		assert.Equal(t, "1.26", got.GoVersion)
		assert.Equal(t, 8, got.NumCPU)
		assert.Equal(t, 100, got.NumGoroutine)
		assert.Equal(t, uint64(5000), got.Memory.AllocBytes)
		assert.Equal(t, uint32(50), got.GC.NumGC)
		assert.Equal(t, "2025-01-01", got.GC.LastGC)
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()
		got := convertRuntime(map[string]any{})
		assert.Equal(t, Runtime{}, got)
	})

	t.Run("string only", func(t *testing.T) {
		t.Parallel()
		data := map[string]any{"go_version": "1.27"}
		got := convertRuntime(data)
		assert.Equal(t, "1.27", got.GoVersion)
		assert.Equal(t, 0, got.NumCPU)
	})
}

func TestConvertRuntimeMemory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		want RuntimeMemory
	}{
		{
			name: "full memory data",
			data: map[string]any{
				"alloc_bytes":       float64(1000000),
				"total_alloc_bytes": float64(5000000),
				"sys_bytes":         float64(20000000),
				"heap_alloc_bytes":  float64(800000),
				"heap_sys_bytes":    float64(3000000),
				"alloc_mb":          float64(0.95),
				"sys_mb":            float64(19.07),
			},
			want: RuntimeMemory{
				AllocBytes:      1000000,
				TotalAllocBytes: 5000000,
				SysBytes:        20000000,
				HeapAllocBytes:  800000,
				HeapSysBytes:    3000000,
				AllocMB:         0.95,
				SysMB:           19.07,
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
			want: RuntimeMemory{},
		},
		{
			name: "partial data",
			data: map[string]any{"alloc_bytes": float64(500), "alloc_mb": float64(0.01)},
			want: RuntimeMemory{AllocBytes: 500, AllocMB: 0.01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertRuntimeMemory(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertGC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		want GC
	}{
		{
			name: "full gc data",
			data: map[string]any{
				"num_gc":         float64(200),
				"pause_total_ns": float64(80000000),
				"last_gc":        "2025-01-01T01:00:00Z",
			},
			want: GC{
				NumGC:        200,
				PauseTotalNs: 80000000,
				LastGC:       "2025-01-01T01:00:00Z",
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
			want: GC{},
		},
		{
			name: "last_gc only",
			data: map[string]any{"last_gc": "recent"},
			want: GC{LastGC: "recent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertGC(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertBuildInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		want BuildInfo
	}{
		{
			name: "full build info",
			data: map[string]any{
				"go_version":   "1.26",
				"main_module":  "flowbot",
				"main_version": "v1.0.0",
				"settings": map[string]any{
					"CGO_ENABLED": "1",
					"GOOS":        "linux",
				},
			},
			want: BuildInfo{
				GoVersion:   "1.26",
				MainModule:  "flowbot",
				MainVersion: "v1.0.0",
				Settings:    types.KV{"CGO_ENABLED": "1", "GOOS": "linux"},
			},
		},
		{
			name: "empty data",
			data: map[string]any{},
			want: BuildInfo{},
		},
		{
			name: "no settings",
			data: map[string]any{"go_version": "1.26", "main_module": "app"},
			want: BuildInfo{GoVersion: "1.26", MainModule: "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertBuildInfo(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetStringFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		key  string
		want string
	}{
		{
			name: "key exists with string",
			data: map[string]any{"name": "value"},
			key:  "name",
			want: "value",
		},
		{
			name: "key exists with non-string",
			data: map[string]any{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "key missing",
			data: map[string]any{},
			key:  "missing",
			want: "",
		},
		{
			name: "nil map",
			data: nil,
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getStringFromMap(tt.data, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetFloat64FromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]any
		key  string
		want float64
	}{
		{
			name: "key exists with float64",
			data: map[string]any{"value": 3.14},
			key:  "value",
			want: 3.14,
		},
		{
			name: "key exists with int",
			data: map[string]any{"value": 42},
			key:  "value",
			want: 0,
		},
		{
			name: "key missing",
			data: map[string]any{},
			key:  "value",
			want: 0,
		},
		{
			name: "nil map",
			data: nil,
			key:  "key",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getFloat64FromMap(tt.data, tt.key)
			assert.InDelta(t, tt.want, got, 0)
		})
	}
}
