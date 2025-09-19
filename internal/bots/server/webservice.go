package server

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/shirou/gopsutil/v4/process"
)

var webserviceRules = []webservice.Rule{
	webservice.Post("/upload", upload),
	webservice.Get("/stacktrace", stacktrace),
}

// upload PicGO upload api
//
//	@Summary	upload PicGO upload api
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/server/upload [post]
func upload(ctx fiber.Ctx) error {
	result := make([]string, 0)
	if form, err := ctx.MultipartForm(); err == nil {
		for _, file := range form.File {
			for _, part := range file {
				mimeType := part.Header.Get("Content-Type")
				if !utils.ValidImageContentType(mimeType) {
					continue
				}

				f, err := part.Open()
				if err != nil {
					flog.Error(fmt.Errorf("error opening file: %s, %w", part.Filename, err))
					continue
				}

				url, _, err := store.FileSystem.Upload(&types.FileDef{
					ObjHeader: types.ObjHeader{
						Id: types.Id(),
					},
					MimeType: mimeType,
					Size:     part.Size,
					Location: "/image",
				}, f)
				if err != nil {
					flog.Error(fmt.Errorf("error uploading file: %s, %w", part.Filename, err))
					continue
				}

				result = append(result, url)
			}
		}
	}

	return ctx.JSON(types.KV{
		"success": len(result) > 0,
		"result":  result,
	})
}

// stacktrace get server or goroutines stacktrace
//
//	@Summary	get server or goroutines stacktrace
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/server/stacktrace [get]
func stacktrace(ctx fiber.Ctx) error {
	pid := os.Getpid()
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		flog.Error(fmt.Errorf("failed to get process info: %w", err))
		return ctx.JSON(types.KV{"error": "failed to get process info"})
	}

	// Gather basic process information
	processInfo := types.KV{}

	if name, err := proc.Name(); err == nil {
		processInfo["name"] = name
	}

	processInfo["pid"] = pid

	if createTime, err := proc.CreateTime(); err == nil {
		processInfo["create_time"] = time.Unix(createTime/1000, 0).Format(time.RFC3339)
		processInfo["uptime_seconds"] = time.Since(time.Unix(createTime/1000, 0)).Seconds()
	}

	if memInfo, err := proc.MemoryInfo(); err == nil {
		processInfo["memory"] = types.KV{
			"rss_bytes":  memInfo.RSS,
			"vms_bytes":  memInfo.VMS,
			"swap_bytes": memInfo.Swap,
			"rss_mb":     float64(memInfo.RSS) / 1024 / 1024,
			"vms_mb":     float64(memInfo.VMS) / 1024 / 1024,
		}
	}

	if cpuPercent, err := proc.CPUPercent(); err == nil {
		processInfo["cpu_percent"] = cpuPercent
	}

	if numThreads, err := proc.NumThreads(); err == nil {
		processInfo["num_threads"] = numThreads
	}

	// Gather Go runtime information
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	runtimeInfo := types.KV{
		"go_version":    runtime.Version(),
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
		"memory": types.KV{
			"alloc_bytes":         memStats.Alloc,
			"total_alloc_bytes":   memStats.TotalAlloc,
			"sys_bytes":           memStats.Sys,
			"heap_alloc_bytes":    memStats.HeapAlloc,
			"heap_sys_bytes":      memStats.HeapSys,
			"heap_inuse_bytes":    memStats.HeapInuse,
			"heap_released_bytes": memStats.HeapReleased,
			"stack_inuse_bytes":   memStats.StackInuse,
			"stack_sys_bytes":     memStats.StackSys,
			"alloc_mb":            float64(memStats.Alloc) / 1024 / 1024,
			"sys_mb":              float64(memStats.Sys) / 1024 / 1024,
		},
		"gc": types.KV{
			"num_gc":         memStats.NumGC,
			"pause_total_ns": memStats.PauseTotalNs,
			"last_gc":        time.Unix(0, int64(memStats.LastGC)).Format(time.RFC3339),
		},
	}

	// Capture goroutines stack traces
	stackBuf := make([]byte, 1024*1024*10) // 10MB buffer
	stackSize := runtime.Stack(stackBuf, true)
	stackTrace := string(stackBuf[:stackSize])

	// Gather build information
	buildInfo := types.KV{}
	if info, ok := debug.ReadBuildInfo(); ok {
		buildInfo["go_version"] = info.GoVersion
		buildInfo["main_module"] = info.Main.Path
		if info.Main.Version != "" {
			buildInfo["main_version"] = info.Main.Version
		}

		settings := types.KV{}
		for _, setting := range info.Settings {
			settings[setting.Key] = setting.Value
		}
		buildInfo["settings"] = settings
	}

	result := types.KV{
		"timestamp":   time.Now().Format(time.RFC3339),
		"process":     processInfo,
		"runtime":     runtimeInfo,
		"build_info":  buildInfo,
		"stack_trace": stackTrace,
	}

	return ctx.JSON(result)
}
