package client

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// ServerClient provides access to the server management API.
type ServerClient struct {
	c *Client
}

// UploadResult contains the result of uploading files.
type UploadResult struct {
	Success bool     `json:"success"`
	URLs    []string `json:"result"`
}

// Upload uploads files to the server.
// files should be a map of field name to file content.
func (s *ServerClient) Upload(ctx context.Context, files map[string]io.Reader, filenames map[string]string) (*UploadResult, error) {
	if err := validateUploadFiles(files, filenames); err != nil {
		return nil, err
	}

	req := s.c.RawRequest().SetContext(ctx)

	for fieldName, file := range files {
		filename := filenames[fieldName]
		if filename == "" {
			filename = fieldName
		}
		req.SetFileReader(fieldName, filename, file)
	}

	resp, err := req.Post("/service/server/upload")
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	var result UploadResult
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func validateUploadFiles(files map[string]io.Reader, filenames map[string]string) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one file is required")
	}
	if len(files) > validate.MaxFileCount {
		return fmt.Errorf("file count exceeds maximum of %d", validate.MaxFileCount)
	}
	for fieldName := range files {
		if fieldName == "" {
			return fmt.Errorf("field name cannot be empty")
		}
		if len(fieldName) > validate.NameMaxLen {
			return fmt.Errorf("field name exceeds maximum length of %d", validate.NameMaxLen)
		}
		if filename, ok := filenames[fieldName]; ok {
			if len(filename) > validate.NameMaxLen {
				return fmt.Errorf("filename exceeds maximum length of %d", validate.NameMaxLen)
			}
		}
	}
	return nil
}

// UploadMultipart uploads files using multipart form data.
func (s *ServerClient) UploadMultipart(ctx context.Context, writer *multipart.Writer, body io.Reader) (*UploadResult, error) {
	req := s.c.RawRequest().SetContext(ctx).
		SetHeader("Content-Type", writer.FormDataContentType()).
		SetBody(body)

	resp, err := req.Post("/service/server/upload")
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	var result UploadResult
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StacktraceResult contains server diagnostic information.
type StacktraceResult struct {
	Timestamp  string    `json:"timestamp"`
	Process    Process   `json:"process"`
	Runtime    Runtime   `json:"runtime"`
	BuildInfo  BuildInfo `json:"build_info"`
	StackTrace string    `json:"stack_trace"`
}

// Process contains process-level information.
type Process struct {
	Name          string  `json:"name"`
	PID           int     `json:"pid"`
	CreateTime    string  `json:"create_time"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	Memory        Memory  `json:"memory"`
	CPUPercent    float64 `json:"cpu_percent"`
	NumThreads    int32   `json:"num_threads"`
}

// Memory contains memory statistics.
type Memory struct {
	RSSBytes  int64   `json:"rss_bytes"`
	VMSBytes  int64   `json:"vms_bytes"`
	SwapBytes int64   `json:"swap_bytes"`
	RSSMB     float64 `json:"rss_mb"`
	VMSMB     float64 `json:"vms_mb"`
}

// Runtime contains Go runtime information.
type Runtime struct {
	GoVersion    string        `json:"go_version"`
	NumCPU       int           `json:"num_cpu"`
	NumGoroutine int           `json:"num_goroutine"`
	Memory       RuntimeMemory `json:"memory"`
	GC           GC            `json:"gc"`
}

// RuntimeMemory contains Go runtime memory statistics.
type RuntimeMemory struct {
	AllocBytes      uint64  `json:"alloc_bytes"`
	TotalAllocBytes uint64  `json:"total_alloc_bytes"`
	SysBytes        uint64  `json:"sys_bytes"`
	HeapAllocBytes  uint64  `json:"heap_alloc_bytes"`
	HeapSysBytes    uint64  `json:"heap_sys_bytes"`
	AllocMB         float64 `json:"alloc_mb"`
	SysMB           float64 `json:"sys_mb"`
}

// GC contains garbage collection statistics.
type GC struct {
	NumGC        uint32 `json:"num_gc"`
	PauseTotalNs uint64 `json:"pause_total_ns"`
	LastGC       string `json:"last_gc"`
}

// BuildInfo contains build information.
type BuildInfo struct {
	GoVersion   string   `json:"go_version"`
	MainModule  string   `json:"main_module"`
	MainVersion string   `json:"main_version,omitempty"`
	Settings    types.KV `json:"settings"`
}

// Stacktrace retrieves server diagnostic information including goroutine stack traces.
func (s *ServerClient) Stacktrace(ctx context.Context) (*StacktraceResult, error) {
	var result types.KV
	err := s.c.Get(ctx, "/service/server/stacktrace", &result)
	if err != nil {
		return nil, err
	}

	return convertStacktraceResult(result), nil
}

func convertStacktraceResult(data types.KV) *StacktraceResult {
	result := &StacktraceResult{
		Timestamp:  stringOr(data, "timestamp", ""),
		StackTrace: stringOr(data, "stack_trace", ""),
	}

	if proc, ok := data["process"].(map[string]any); ok {
		result.Process = convertProcess(proc)
	}
	if rt, ok := data["runtime"].(map[string]any); ok {
		result.Runtime = convertRuntime(rt)
	}
	if bi, ok := data["build_info"].(map[string]any); ok {
		result.BuildInfo = convertBuildInfo(bi)
	}

	return result
}

func convertProcess(data map[string]any) Process {
	p := Process{
		Name:       getStringFromMap(data, "name"),
		CreateTime: getStringFromMap(data, "create_time"),
	}
	if v, ok := data["pid"].(float64); ok {
		p.PID = int(v)
	}
	if v, ok := data["uptime_seconds"].(float64); ok {
		p.UptimeSeconds = v
	}
	if v, ok := data["cpu_percent"].(float64); ok {
		p.CPUPercent = v
	}
	if v, ok := data["num_threads"].(float64); ok {
		p.NumThreads = int32(v)
	}
	if mem, ok := data["memory"].(map[string]any); ok {
		p.Memory = convertMemory(mem)
	}
	return p
}

func convertMemory(data map[string]any) Memory {
	m := Memory{}
	if v, ok := data["rss_bytes"].(float64); ok {
		m.RSSBytes = int64(v)
	}
	if v, ok := data["vms_bytes"].(float64); ok {
		m.VMSBytes = int64(v)
	}
	if v, ok := data["swap_bytes"].(float64); ok {
		m.SwapBytes = int64(v)
	}
	if v, ok := data["rss_mb"].(float64); ok {
		m.RSSMB = v
	}
	if v, ok := data["vms_mb"].(float64); ok {
		m.VMSMB = v
	}
	return m
}

func convertRuntime(data map[string]any) Runtime {
	r := Runtime{
		GoVersion:    getStringFromMap(data, "go_version"),
		NumCPU:       int(getFloat64FromMap(data, "num_cpu")),
		NumGoroutine: int(getFloat64FromMap(data, "num_goroutine")),
	}
	if mem, ok := data["memory"].(map[string]any); ok {
		r.Memory = convertRuntimeMemory(mem)
	}
	if gc, ok := data["gc"].(map[string]any); ok {
		r.GC = convertGC(gc)
	}
	return r
}

func convertRuntimeMemory(data map[string]any) RuntimeMemory {
	m := RuntimeMemory{}
	if v, ok := data["alloc_bytes"].(float64); ok {
		m.AllocBytes = uint64(v)
	}
	if v, ok := data["total_alloc_bytes"].(float64); ok {
		m.TotalAllocBytes = uint64(v)
	}
	if v, ok := data["sys_bytes"].(float64); ok {
		m.SysBytes = uint64(v)
	}
	if v, ok := data["heap_alloc_bytes"].(float64); ok {
		m.HeapAllocBytes = uint64(v)
	}
	if v, ok := data["heap_sys_bytes"].(float64); ok {
		m.HeapSysBytes = uint64(v)
	}
	if v, ok := data["alloc_mb"].(float64); ok {
		m.AllocMB = v
	}
	if v, ok := data["sys_mb"].(float64); ok {
		m.SysMB = v
	}
	return m
}

func convertGC(data map[string]any) GC {
	g := GC{
		LastGC: getStringFromMap(data, "last_gc"),
	}
	if v, ok := data["num_gc"].(float64); ok {
		g.NumGC = uint32(v)
	}
	if v, ok := data["pause_total_ns"].(float64); ok {
		g.PauseTotalNs = uint64(v)
	}
	return g
}

func convertBuildInfo(data map[string]any) BuildInfo {
	b := BuildInfo{
		GoVersion:   getStringFromMap(data, "go_version"),
		MainModule:  getStringFromMap(data, "main_module"),
		MainVersion: getStringFromMap(data, "main_version"),
	}
	if settings, ok := data["settings"].(map[string]any); ok {
		b.Settings = settings
	}
	return b
}

func getStringFromMap(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getFloat64FromMap(data map[string]any, key string) float64 {
	if v, ok := data[key].(float64); ok {
		return v
	}
	return 0
}
