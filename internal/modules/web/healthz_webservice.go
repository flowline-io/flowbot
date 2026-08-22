package web

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

const (
	healthzSnapshotTTL      = 30 * time.Second
	healthzCapCacheTTL      = 30 * time.Second
	healthzCapLookupTimeout = time.Second
)

var (
	healthzSnapshotMu sync.Mutex
	healthzSnapshot   partials.HealthzData
	healthzSnapshotAt time.Time
	healthzRefreshMu  sync.Mutex
	healthzRefreshing bool
	healthzRefreshWG  sync.WaitGroup

	healthzCapMu         sync.Mutex
	healthzCapSnapshot   []partials.HealthzCap
	healthzCapSnapshotAt time.Time
	healthzCapRefreshMu  sync.Mutex
	healthzCapRefreshing bool
	healthzCapRefreshWG  sync.WaitGroup
)

var healthzWebserviceRules = []webservice.Rule{
	webservice.Get("/healthz", healthzPage),
	webservice.Get("/healthz/capabilities", healthzCapabilitiesPartial),
}

// healthzPage renders the system health dashboard (infra metrics only; capabilities load async).
func healthzPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	hctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data := gatherInfraHealthzData(hctx)

	ctx.Type("html")
	if ctx.Get("HX-Request") != "" {
		return partials.HealthzStatus(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	return pages.HealthzPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// healthzCapabilitiesPartial returns the capability health table for deferred HTMX loads.
func healthzCapabilitiesPartial(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	hctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	caps := gatherCapabilityHealth(hctx)
	ctx.Type("html")
	return partials.HealthzCapabilities(ctx.Context(), caps).Render(ctx.Context(), ctx.Response().BodyWriter())
}

// gatherInfraHealthzData returns infra metrics with stale-while-revalidate caching.
func gatherInfraHealthzData(ctx context.Context) partials.HealthzData {
	healthzSnapshotMu.Lock()
	has := !healthzSnapshotAt.IsZero()
	fresh := has && time.Since(healthzSnapshotAt) < healthzSnapshotTTL
	data := healthzSnapshot
	healthzSnapshotMu.Unlock()

	if fresh {
		return data
	}
	if has {
		triggerInfraHealthzRefresh()
		return data
	}

	data = collectInfraHealthzData(ctx)
	storeInfraHealthzSnapshot(data)
	return data
}

func triggerInfraHealthzRefresh() {
	healthzRefreshMu.Lock()
	if healthzRefreshing {
		healthzRefreshMu.Unlock()
		return
	}
	healthzRefreshing = true
	healthzRefreshWG.Add(1)
	healthzRefreshMu.Unlock()

	go func() {
		defer func() {
			healthzRefreshMu.Lock()
			healthzRefreshing = false
			healthzRefreshMu.Unlock()
			healthzRefreshWG.Done()
		}()
		rctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		storeInfraHealthzSnapshot(collectInfraHealthzData(rctx))
	}()
}

func storeInfraHealthzSnapshot(data partials.HealthzData) {
	healthzSnapshotMu.Lock()
	defer healthzSnapshotMu.Unlock()
	healthzSnapshot = data
	healthzSnapshotAt = time.Now()
}

// gatherCapabilityHealth returns capability statuses with stale-while-revalidate caching.
func gatherCapabilityHealth(ctx context.Context) []partials.HealthzCap {
	healthzCapMu.Lock()
	has := !healthzCapSnapshotAt.IsZero()
	fresh := has && time.Since(healthzCapSnapshotAt) < healthzCapCacheTTL
	caps := append([]partials.HealthzCap(nil), healthzCapSnapshot...)
	healthzCapMu.Unlock()

	if fresh {
		return caps
	}
	if has {
		triggerCapabilityHealthRefresh()
		return caps
	}

	caps = collectCapabilityHealth(ctx)
	storeCapabilityHealthSnapshot(caps)
	return caps
}

func triggerCapabilityHealthRefresh() {
	healthzCapRefreshMu.Lock()
	if healthzCapRefreshing {
		healthzCapRefreshMu.Unlock()
		return
	}
	healthzCapRefreshing = true
	healthzCapRefreshWG.Add(1)
	healthzCapRefreshMu.Unlock()

	go func() {
		defer func() {
			healthzCapRefreshMu.Lock()
			healthzCapRefreshing = false
			healthzCapRefreshMu.Unlock()
			healthzCapRefreshWG.Done()
		}()
		rctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		storeCapabilityHealthSnapshot(collectCapabilityHealth(rctx))
	}()
}

func storeCapabilityHealthSnapshot(caps []partials.HealthzCap) {
	healthzCapMu.Lock()
	defer healthzCapMu.Unlock()
	healthzCapSnapshot = append([]partials.HealthzCap(nil), caps...)
	healthzCapSnapshotAt = time.Now()
}

// collectInfraHealthzData collects Postgres/Redis/runtime/error metrics (no capability probes).
func collectInfraHealthzData(ctx context.Context) partials.HealthzData {
	data := partials.HealthzData{}

	var (
		pgLatency, redisLatency time.Duration
		pgOk, redisOk           bool
		wg                      sync.WaitGroup
	)

	wg.Go(func() {
		if store.Database != nil && store.Database.IsOpen() {
			latency, err := store.Database.Ping(ctx)
			pgLatency = latency
			pgOk = err == nil
		}
	})

	wg.Go(func() {
		if rs := cache.DefaultRedisStore(); rs != nil {
			latency, err := rs.Ping(ctx)
			redisLatency = latency
			redisOk = err == nil
		}
	})

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	data.Goroutines = runtime.NumGoroutine()
	data.HeapAlloc = memStats.HeapAlloc
	data.TotalAlloc = memStats.TotalAlloc
	data.SysMem = memStats.Sys
	data.NumGC = memStats.NumGC
	if memStats.NumGC > 0 {
		pauseNs := memStats.PauseNs[(memStats.NumGC+255)%256]
		if pause, ok := utils.Uint64ToInt64(pauseNs); ok {
			data.LastGCPause = time.Duration(pause)
		}
	}

	wg.Wait()

	data.PostgresLatency = pgLatency
	data.PostgresOk = pgOk
	data.RedisLatency = redisLatency
	data.RedisOk = redisOk

	allErrors := flog.RecentErrors()
	start := 0
	if len(allErrors) > 10 {
		start = len(allErrors) - 10
	}
	data.Errors = allErrors[start:]

	return data
}

func collectCapabilityHealth(ctx context.Context) []partials.HealthzCap {
	descriptors := hub.Default.List()
	caps := make([]partials.HealthzCap, len(descriptors))
	var wg sync.WaitGroup

	for i, desc := range descriptors {
		wg.Go(func() {
			capCtx, cancel := context.WithTimeout(ctx, healthzCapLookupTimeout)
			defer cancel()

			info := partials.HealthzCap{
				Type: string(desc.Type),
			}

			result, err := capability.Invoke(capCtx, desc.Type, "health", map[string]any{})
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
			caps[i] = info
		})
	}
	wg.Wait()
	return caps
}

func waitHealthzRefresh() {
	healthzRefreshWG.Wait()
	healthzCapRefreshWG.Wait()
}

func clearHealthzSnapshot() {
	waitHealthzRefresh()

	healthzSnapshotMu.Lock()
	healthzSnapshot = partials.HealthzData{}
	healthzSnapshotAt = time.Time{}
	healthzSnapshotMu.Unlock()

	healthzCapMu.Lock()
	healthzCapSnapshot = nil
	healthzCapSnapshotAt = time.Time{}
	healthzCapMu.Unlock()
}

// gatherHealthzData is retained for tests that assert snapshot caching of infra metrics.
func gatherHealthzData(ctx context.Context) partials.HealthzData {
	return gatherInfraHealthzData(ctx)
}
