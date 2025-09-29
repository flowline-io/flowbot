package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/fx"
)

var (
	// Go runtime memory metrics
	memoryAllocated = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_memory_allocated_bytes",
		Help: "Current allocated memory in bytes.",
	})

	memoryHeapInuse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_memory_heap_inuse_bytes",
		Help: "Heap memory currently in use in bytes.",
	})

	memoryHeapSys = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_memory_heap_sys_bytes",
		Help: "Heap memory obtained from system in bytes.",
	})

	// Number of goroutines
	goroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_goroutines_count",
		Help: "Current number of goroutines.",
	})

	// GC related metrics
	gcPauseTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_gc_pause_total_ns",
		Help: "Total GC pause time in nanoseconds.",
	})

	gcNumGC = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_gc_num_gc_total",
		Help: "Total number of GC runs.",
	})

	// Number of CPUs
	cpuCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "golang_runtime_cpu_count",
		Help: "Number of logical CPUs usable by the current process.",
	})
)

func doPush(hostid, hostname string) {
	// Get Go runtime statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Update memory-related metrics
	memoryAllocated.Set(float64(memStats.Alloc))
	memoryHeapInuse.Set(float64(memStats.HeapInuse))
	memoryHeapSys.Set(float64(memStats.HeapSys))

	// Update goroutine count
	goroutines.Set(float64(runtime.NumGoroutine()))

	// Update GC-related metrics
	gcPauseTotal.Set(float64(memStats.PauseTotalNs))
	gcNumGC.Set(float64(memStats.NumGC))

	// Update CPU count
	cpuCount.Set(float64(runtime.NumCPU()))

	// Create a Pusher to push metrics to the Pushgateway
	// Grouping adds labels; 'job' is required, 'instance' distinguishes different instances
	pusher := push.New(config.App.Metrics.Endpoint, "flowbot-agent").
		Grouping("instance", hostid).   // instance label
		Grouping("hostname", hostname). // hostname label
		Collector(memoryAllocated).
		Collector(memoryHeapInuse).
		Collector(memoryHeapSys).
		Collector(goroutines).
		Collector(gcPauseTotal).
		Collector(gcNumGC).
		Collector(cpuCount)

	if err := pusher.Add(); err != nil {
		flog.Error(fmt.Errorf("Could not push golang metrics to Pushgateway: %w", err))
	}
}

func tickMetrics(lc fx.Lifecycle, _ config.Type) {
	if !config.App.Metrics.Enabled {
		flog.Info("Metrics push is disabled.")
		return
	}
	metricsTicker := time.NewTicker(10 * time.Second)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			flog.Info("Metrics push is enabled.")
			hostid, hostname, err := utils.HostInfo()
			if err != nil {
				flog.Error(fmt.Errorf("[metrics] failed to get host info, %w", err))
				return err
			}

			go func() {
				for range metricsTicker.C {
					doPush(hostid, hostname)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			metricsTicker.Stop()
			return nil
		},
	})
}
