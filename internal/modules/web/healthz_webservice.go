package web

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var healthzWebserviceRules = []webservice.Rule{
	webservice.Get("/healthz", healthzPage, route.WithNotAuth()),
}

// healthzPage renders the system health dashboard.
func healthzPage(ctx fiber.Ctx) error {
	hctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := gatherHealthzData(hctx)

	ctx.Type("html")
	if ctx.Get("HX-Request") != "" {
		return partials.HealthzStatus(data).Render(context.Background(), ctx.Response().BodyWriter())
	}
	return pages.HealthzPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

// gatherHealthzData collects all health metrics for the dashboard.
func gatherHealthzData(ctx context.Context) partials.HealthzData {
	data := partials.HealthzData{}

	// PostgreSQL ping
	if store.Database != nil && store.Database.IsOpen() {
		latency, err := store.Database.Ping(ctx)
		data.PostgresLatency = latency
		data.PostgresOk = err == nil
	}

	// Redis ping
	if rs := cache.DefaultRedisStore(); rs != nil {
		latency, err := rs.Ping(ctx)
		data.RedisLatency = latency
		data.RedisOk = err == nil
	}

	// Runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	data.Goroutines = runtime.NumGoroutine()
	data.HeapAlloc = memStats.HeapAlloc
	data.TotalAlloc = memStats.TotalAlloc
	data.SysMem = memStats.Sys
	data.NumGC = memStats.NumGC
	if memStats.NumGC > 0 {
		data.LastGCPause = time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
	}

	// Capability health checks
	descriptors := hub.Default.List()
	caps := make([]partials.HealthzCap, len(descriptors))
	var wg sync.WaitGroup

	for i, desc := range descriptors {
		wg.Add(1)
		go func(idx int, d hub.Descriptor) {
			defer wg.Done()
			capCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			info := partials.HealthzCap{
				Type:    string(d.Type),
				Backend: string(d.Backend),
			}

			result, err := ability.Invoke(capCtx, d.Type, "health", map[string]any{})
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					info.Status = "timeout"
				} else {
					info.Status = "unhealthy"
					info.Error = err.Error()
				}
			} else if result != nil && result.Data != nil {
				if ok, isBool := result.Data.(bool); isBool && ok {
					info.Status = "healthy"
				} else {
					info.Status = "unhealthy"
				}
			} else {
				info.Status = "na"
			}
			caps[idx] = info
		}(i, desc)
	}
	wg.Wait()
	data.Capabilities = caps

	// Recent errors (last 10)
	allErrors := flog.RecentErrors()
	start := 0
	if len(allErrors) > 10 {
		start = len(allErrors) - 10
	}
	data.Errors = allErrors[start:]

	return data
}
