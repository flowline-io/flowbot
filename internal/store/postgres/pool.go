package postgres

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// PoolConfig holds all tunable connection pool parameters.
// Zero or negative values are replaced by defaults in applyDefaults().
type PoolConfig struct {
	MaxOpenConns        int `json:"max_open_conns,omitempty"`
	MaxIdleConns        int `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime     int `json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime     int `json:"conn_max_idle_time,omitempty"`
	HealthCheckInterval int `json:"pool_health_check_interval,omitempty"`
	HealthCheckTimeout  int `json:"pool_health_check_timeout,omitempty"`
}

// Default values when config field is zero or negative.
const (
	defaultMaxOpenConns        = 25
	defaultMaxIdleConns        = 5
	defaultConnMaxLifetime     = 300
	defaultConnMaxIdleTime     = 60
	defaultHealthCheckInterval = 30
	defaultHealthCheckTimeout  = 5
)

// applyDefaults replaces zero or negative PoolConfig fields with defaults.
func (c *PoolConfig) applyDefaults() {
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = defaultMaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = defaultMaxIdleConns
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = defaultConnMaxLifetime
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = defaultConnMaxIdleTime
	}
	if c.HealthCheckInterval < 0 {
		c.HealthCheckInterval = defaultHealthCheckInterval
	}
	if c.HealthCheckTimeout <= 0 {
		c.HealthCheckTimeout = defaultHealthCheckTimeout
	}
}

// ---------------------------------------------------------------------------
// Prometheus metrics
// ---------------------------------------------------------------------------

type poolMetrics struct {
	openConns         prometheus.Gauge
	idleConns         prometheus.Gauge
	inUse             prometheus.Gauge
	waitCount         prometheus.Gauge
	waitDuration      prometheus.Gauge
	maxIdleClosed     prometheus.Gauge
	maxLifetimeClosed prometheus.Gauge
	healthTotal       prometheus.Counter
	healthErrors      prometheus.Counter
}

var (
	poolMetricsInst *poolMetrics
	poolMetricsOnce sync.Once
)

// registerMetrics creates all pool Prometheus metrics exactly once.
func registerMetrics() *poolMetrics {
	poolMetricsOnce.Do(func() {
		poolMetricsInst = &poolMetrics{
			openConns: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_connections_open",
				Help: "Current number of open connections.",
			}),
			idleConns: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_connections_idle",
				Help: "Current number of idle connections.",
			}),
			inUse: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_connections_in_use",
				Help: "Current number of in-use connections.",
			}),
			waitCount: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_wait_count_total",
				Help: "Cumulative number of connections waited for.",
			}),
			waitDuration: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_wait_duration_seconds_total",
				Help: "Cumulative wait time for connections in seconds.",
			}),
			maxIdleClosed: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_max_idle_closed_total",
				Help: "Cumulative number of connections closed due to SetConnMaxIdleTime.",
			}),
			maxLifetimeClosed: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "flowbot_db_pool_max_lifetime_closed_total",
				Help: "Cumulative number of connections closed due to SetConnMaxLifetime.",
			}),
			healthTotal: promauto.NewCounter(prometheus.CounterOpts{
				Name: "flowbot_db_pool_health_check_total",
				Help: "Total number of health check pings performed.",
			}),
			healthErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "flowbot_db_pool_health_check_errors_total",
				Help: "Total number of failed health check pings.",
			}),
		}
	})
	return poolMetricsInst
}

// ---------------------------------------------------------------------------
// PoolManager
// ---------------------------------------------------------------------------

// PoolManager wraps an *sql.DB and manages pool configuration, health
// checking, and Prometheus metric collection.
type PoolManager struct {
	db      *sql.DB
	config  PoolConfig
	metrics *poolMetrics
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewPoolManager creates a PoolManager, applies pool settings to db, and
// registers Prometheus metrics. The returned PoolManager is not started;
// call Start() to begin background health checking.
func NewPoolManager(db *sql.DB, cfg PoolConfig) *PoolManager {
	cfg.applyDefaults()
	pm := &PoolManager{
		db:      db,
		config:  cfg,
		metrics: registerMetrics(),
	}
	pm.applyConfig()
	return pm
}

// applyConfig sets all pool parameters on the underlying *sql.DB.
func (pm *PoolManager) applyConfig() {
	pm.db.SetMaxOpenConns(pm.config.MaxOpenConns)
	pm.db.SetMaxIdleConns(pm.config.MaxIdleConns)
	pm.db.SetConnMaxLifetime(time.Duration(pm.config.ConnMaxLifetime) * time.Second)
	pm.db.SetConnMaxIdleTime(time.Duration(pm.config.ConnMaxIdleTime) * time.Second)
}

// Start begins the background health check pinger. When HealthCheckInterval
// is 0, the pinger is disabled and Start returns immediately.
func (pm *PoolManager) Start(ctx context.Context) {
	if pm.config.HealthCheckInterval <= 0 {
		return
	}
	pm.done = make(chan struct{})
	var pingerCtx context.Context
	pingerCtx, pm.cancel = context.WithCancel(ctx)
	go pm.pingerLoop(pingerCtx)
}

// Stop cancels the background pinger and waits up to 5 seconds for it to
// exit. Safe to call on a PoolManager that was never started or already
// stopped.
func (pm *PoolManager) Stop() {
	if pm.cancel == nil {
		return
	}
	pm.cancel()
	pm.cancel = nil
	if pm.done != nil {
		select {
		case <-pm.done:
		case <-time.After(5 * time.Second):
			flog.Warn("pool manager: stop timed out waiting for pinger")
		}
	}
}

// pingerLoop periodically pings the database and collects pool statistics
// for Prometheus metrics.
func (pm *PoolManager) pingerLoop(ctx context.Context) {
	defer close(pm.done)

	ticker := time.NewTicker(time.Duration(pm.config.HealthCheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.healthCheck(ctx)
			pm.collectStats()
		}
	}
}

// healthCheck pings the database with a configurable timeout and updates
// health metrics. Failures are logged as warnings, not fatal errors.
func (pm *PoolManager) healthCheck(ctx context.Context) {
	pm.metrics.healthTotal.Inc()

	timeout := time.Duration(pm.config.HealthCheckTimeout) * time.Second
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := pm.db.PingContext(healthCtx); err != nil {
		pm.metrics.healthErrors.Inc()
		flog.Warn("pool manager: health check ping failed: %v", err)
	}
}

// collectStats reads sql.DBStats and updates all Prometheus gauges.
func (pm *PoolManager) collectStats() {
	stats := pm.db.Stats()
	pm.metrics.openConns.Set(float64(stats.OpenConnections))
	pm.metrics.idleConns.Set(float64(stats.Idle))
	pm.metrics.inUse.Set(float64(stats.InUse))
	pm.metrics.waitCount.Set(float64(stats.WaitCount))
	pm.metrics.waitDuration.Set(stats.WaitDuration.Seconds())
	pm.metrics.maxIdleClosed.Set(float64(stats.MaxIdleClosed))
	pm.metrics.maxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))
}

// ApplyDefaults applies conservative pool limits to an *sql.DB. Intended for
// test paths and CLI tools where full pool configuration is unnecessary.
// Settings: MaxOpenConns=10, MaxIdleConns=2, Lifetime=120s, IdleTime=30s.
func ApplyDefaults(db *sql.DB) {
	if db == nil {
		return
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(120 * time.Second)
	db.SetConnMaxIdleTime(30 * time.Second)
}
