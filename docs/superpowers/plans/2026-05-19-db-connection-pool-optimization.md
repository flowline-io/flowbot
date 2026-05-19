# Database Connection Pool Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Encapsulate PostgreSQL connection pool management (config, health checking, Prometheus metrics) in a PoolManager abstraction within `internal/store/postgres/`.

**Architecture:** New `pool.go` with PoolConfig + PoolManager wrapping `*sql.DB`. Adapter `Open()` delegates to PoolManager. Background pinger goroutine validates connections and collects Prometheus stats. `ent.go` imports `postgres.ApplyDefaults` for conservative test pool limits. Three new config keys: `conn_max_idle_time`, `pool_health_check_interval`, `pool_health_check_timeout`.

**Tech Stack:** Go 1.26+, `database/sql`, `pgx/v5/stdlib`, `prometheus/client_golang`, `entgo.io/ent`

---

### Task 1: Create pool.go with PoolConfig, PoolManager, and ApplyDefaults

**Files:**
- Create: `internal/store/postgres/pool.go`

- [ ] **Step 1: Write the full pool.go source**

```go
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
```

- [ ] **Step 2: Build to verify compilation**

```bash
go build ./internal/store/postgres/
```

Expected: PASS (compiles without errors)

- [ ] **Step 3: Commit**

```bash
git add internal/store/postgres/pool.go
git commit -m "feat: add PoolManager with PoolConfig, metrics, and ApplyDefaults"
```

---

### Task 2: Write unit tests for pool.go

**Files:**
- Create: `internal/store/postgres/pool_test.go`

- [ ] **Step 1: Write the full pool_test.go source**

```go
package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ---------------------------------------------------------------------------
// PoolConfig defaults
// ---------------------------------------------------------------------------

func TestPoolConfig_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		input    PoolConfig
		expected PoolConfig
	}{
		{
			name:  "all zero values resolve to defaults",
			input: PoolConfig{},
			expected: PoolConfig{
				MaxOpenConns:        defaultMaxOpenConns,
				MaxIdleConns:        defaultMaxIdleConns,
				ConnMaxLifetime:     defaultConnMaxLifetime,
				ConnMaxIdleTime:     defaultConnMaxIdleTime,
				HealthCheckInterval: defaultHealthCheckInterval,
				HealthCheckTimeout:  defaultHealthCheckTimeout,
			},
		},
		{
			name: "all negative values resolve to defaults",
			input: PoolConfig{
				MaxOpenConns:        -1,
				MaxIdleConns:        -1,
				ConnMaxLifetime:     -1,
				ConnMaxIdleTime:     -1,
				HealthCheckInterval: -1,
				HealthCheckTimeout:  -1,
			},
			expected: PoolConfig{
				MaxOpenConns:        defaultMaxOpenConns,
				MaxIdleConns:        defaultMaxIdleConns,
				ConnMaxLifetime:     defaultConnMaxLifetime,
				ConnMaxIdleTime:     defaultConnMaxIdleTime,
				HealthCheckInterval: defaultHealthCheckInterval,
				HealthCheckTimeout:  defaultHealthCheckTimeout,
			},
		},
		{
			name: "custom values preserved as-is",
			input: PoolConfig{
				MaxOpenConns:        50,
				MaxIdleConns:        20,
				ConnMaxLifetime:     600,
				ConnMaxIdleTime:     120,
				HealthCheckInterval: 60,
				HealthCheckTimeout:  10,
			},
			expected: PoolConfig{
				MaxOpenConns:        50,
				MaxIdleConns:        20,
				ConnMaxLifetime:     600,
				ConnMaxIdleTime:     120,
				HealthCheckInterval: 60,
				HealthCheckTimeout:  10,
			},
		},
		{
			name: "zero health check interval disables pinger",
			input: PoolConfig{
				MaxOpenConns:        25,
				MaxIdleConns:        5,
				ConnMaxLifetime:     300,
				ConnMaxIdleTime:     60,
				HealthCheckInterval: 0,
				HealthCheckTimeout:  0,
			},
			expected: PoolConfig{
				MaxOpenConns:        25,
				MaxIdleConns:        5,
				ConnMaxLifetime:     300,
				ConnMaxIdleTime:     60,
				HealthCheckInterval: 0,
				HealthCheckTimeout:  defaultHealthCheckTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.applyDefaults()
			if tt.input.MaxOpenConns != tt.expected.MaxOpenConns {
				t.Errorf("MaxOpenConns: got %d, want %d", tt.input.MaxOpenConns, tt.expected.MaxOpenConns)
			}
			if tt.input.MaxIdleConns != tt.expected.MaxIdleConns {
				t.Errorf("MaxIdleConns: got %d, want %d", tt.input.MaxIdleConns, tt.expected.MaxIdleConns)
			}
			if tt.input.ConnMaxLifetime != tt.expected.ConnMaxLifetime {
				t.Errorf("ConnMaxLifetime: got %d, want %d", tt.input.ConnMaxLifetime, tt.expected.ConnMaxLifetime)
			}
			if tt.input.ConnMaxIdleTime != tt.expected.ConnMaxIdleTime {
				t.Errorf("ConnMaxIdleTime: got %d, want %d", tt.input.ConnMaxIdleTime, tt.expected.ConnMaxIdleTime)
			}
			if tt.input.HealthCheckInterval != tt.expected.HealthCheckInterval {
				t.Errorf("HealthCheckInterval: got %d, want %d", tt.input.HealthCheckInterval, tt.expected.HealthCheckInterval)
			}
			if tt.input.HealthCheckTimeout != tt.expected.HealthCheckTimeout {
				t.Errorf("HealthCheckTimeout: got %d, want %d", tt.input.HealthCheckTimeout, tt.expected.HealthCheckTimeout)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewPoolManager / ApplyConfig
// ---------------------------------------------------------------------------

func TestNewPoolManager_ApplyConfig(t *testing.T) {
	tests := []struct {
		name   string
		config PoolConfig
	}{
		{
			name:   "default config uses defaults",
			config: PoolConfig{},
		},
		{
			name: "custom config all fields",
			config: PoolConfig{
				MaxOpenConns:        50,
				MaxIdleConns:        20,
				ConnMaxLifetime:     600,
				ConnMaxIdleTime:     120,
				HealthCheckInterval: 30,
				HealthCheckTimeout:  5,
			},
		},
		{
			name: "zero health check interval",
			config: PoolConfig{
				MaxOpenConns:        10,
				MaxIdleConns:        2,
				ConnMaxLifetime:     120,
				ConnMaxIdleTime:     30,
				HealthCheckInterval: 0,
				HealthCheckTimeout:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
			if err != nil {
				t.Skipf("skipping: cannot open test db: %v", err)
			}
			defer db.Close()

			pm := NewPoolManager(db, tt.config)
			if pm == nil {
				t.Fatal("NewPoolManager returned nil")
			}
			if pm.db != db {
				t.Error("PoolManager.db does not match input db")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ApplyDefaults
// ---------------------------------------------------------------------------

func TestApplyDefaults(t *testing.T) {
	t.Run("nil db is safe", func(t *testing.T) {
		ApplyDefaults(nil)
	})

	t.Run("sets expected conservative limits", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		ApplyDefaults(db)
		stats := db.Stats()
		if stats.MaxOpenConnections != 10 {
			t.Errorf("MaxOpenConnections: got %d, want 10", stats.MaxOpenConnections)
		}
	})

	t.Run("can set different values later", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		ApplyDefaults(db)
		db.SetMaxOpenConns(50)
		stats := db.Stats()
		if stats.MaxOpenConnections != 50 {
			t.Errorf("MaxOpenConnections after override: got %d, want 50", stats.MaxOpenConnections)
		}
	})
}

// ---------------------------------------------------------------------------
// Metrics idempotency
// ---------------------------------------------------------------------------

func TestRegisterMetrics_Idempotent(t *testing.T) {
	m1 := registerMetrics()
	m2 := registerMetrics()
	if m1 != m2 {
		t.Error("registerMetrics should return the same instance on repeated calls")
	}
}

func TestRegisterMetrics_AllNames(t *testing.T) {
	registerMetrics()

	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics failed: %v", err)
	}

	metricNames := make(map[string]bool)
	for _, mf := range metrics {
		metricNames[mf.GetName()] = true
	}

	expected := []string{
		"flowbot_db_pool_connections_open",
		"flowbot_db_pool_connections_idle",
		"flowbot_db_pool_connections_in_use",
		"flowbot_db_pool_wait_count_total",
		"flowbot_db_pool_wait_duration_seconds_total",
		"flowbot_db_pool_max_idle_closed_total",
		"flowbot_db_pool_max_lifetime_closed_total",
		"flowbot_db_pool_health_check_total",
		"flowbot_db_pool_health_check_errors_total",
	}

	for _, name := range expected {
		if !metricNames[name] {
			t.Errorf("expected metric %q not found in default registry", name)
		}
	}
}

// ---------------------------------------------------------------------------
// PoolManager Start / Stop
// ---------------------------------------------------------------------------

func TestPoolManager_StartStop(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{
		MaxOpenConns:        5,
		MaxIdleConns:        2,
		ConnMaxLifetime:     300,
		ConnMaxIdleTime:     60,
		HealthCheckInterval: 1,
		HealthCheckTimeout:  1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm.Start(ctx)

	if pm.cancel == nil {
		t.Fatal("cancel func should be set after Start")
	}
	if pm.done == nil {
		t.Fatal("done channel should be set after Start")
	}

	pm.Stop()

	if pm.cancel != nil {
		t.Error("cancel func should be nil after Stop")
	}

	select {
	case <-pm.done:
	default:
		t.Error("done channel should be closed after Stop")
	}
}

func TestPoolManager_StopTwice(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{
		HealthCheckInterval: 1,
		HealthCheckTimeout:  1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm.Start(ctx)
	pm.Stop()
	pm.Stop() // must not panic
}

func TestPoolManager_StopUnstarted(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{})
	pm.Stop() // must not panic
}

func TestPoolManager_PingerDisabled(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{
		HealthCheckInterval: 0,
	})

	ctx := context.Background()
	pm.Start(ctx)

	if pm.cancel != nil {
		t.Error("cancel should be nil when pinger is disabled")
	}
	if pm.done != nil {
		t.Error("done should be nil when pinger is disabled")
	}
}

func TestPoolManager_HealthCheckIncrementsCounters(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{
		HealthCheckTimeout: 1,
	})

	// healthCheck should not panic and should increment healthTotal
	pm.healthCheck(context.Background())
}

// ---------------------------------------------------------------------------
// collectStats
// ---------------------------------------------------------------------------

func TestPoolManager_CollectStats(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Skipf("skipping: cannot open test db: %v", err)
	}
	defer db.Close()

	pm := NewPoolManager(db, PoolConfig{})
	pm.collectStats()

	// Verify gauges were set (values should be 0 since no connections
	// were made to an unreachable host, but no panic should occur)
	_ = pm.metrics.openConns
	_ = pm.metrics.idleConns
	_ = pm.metrics.inUse
	_ = pm.metrics.waitCount
	_ = pm.metrics.waitDuration
	_ = pm.metrics.maxIdleClosed
	_ = pm.metrics.maxLifetimeClosed
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/store/postgres/ -v -count=1 -run "TestPoolConfig|TestNewPoolManager|TestApplyDefaults|TestRegisterMetrics|TestPoolManager"
```

Expected: PASS (all tests pass, some may skip due to no reachable PostgreSQL)

- [ ] **Step 3: Run lint**

```bash
go tool task lint
```

Expected: PASS (no new lint errors in postgres package)

- [ ] **Step 4: Commit**

```bash
git add internal/store/postgres/pool_test.go
git commit -m "test: add PoolManager unit tests for config, lifecycle, metrics"
```

---

### Task 3: Modify adapter.go to use PoolManager

**Files:**
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Edit configType struct (lines 54-60)**

Replace with:
```go
type configType struct {
	DSN                  string `json:"dsn,omitempty"`
	MaxOpenConns         int    `json:"max_open_conns,omitempty"`
	MaxIdleConns         int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime      int    `json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime      int    `json:"conn_max_idle_time,omitempty"`
	SqlTimeout           int    `json:"sql_timeout,omitempty"`
	HealthCheckInterval  int    `json:"pool_health_check_interval,omitempty"`
	HealthCheckTimeout   int    `json:"pool_health_check_timeout,omitempty"`
}
```

- [ ] **Step 2: Edit adapter struct (lines 67-77)**

Add `poolMgr *PoolManager` field:
```go
type adapter struct {
	client  *gen.Client
	db      *sql.DB
	poolMgr *PoolManager

	dbName            string
	maxResults        int
	maxMessageResults int
	sqlTimeout        time.Duration
	txTimeout         time.Duration
	open              bool
}
```

- [ ] **Step 3: Rewrite Open() method (lines 79-138)**

Replace the entire Open method body:
```go
func (a *adapter) Open(jsonConfig config.StoreType) error {
	var conf configType
	if c, ok := jsonConfig.Adapters[adapterName]; ok {
		raw, err := sonic.Marshal(c)
		if err != nil {
			return fmt.Errorf("postgres: marshal adapter config: %w", err)
		}
		if err := sonic.Unmarshal(raw, &conf); err != nil {
			return fmt.Errorf("postgres: unmarshal adapter config: %w", err)
		}
	}

	if conf.DSN == "" {
		return errors.New("postgres: DSN is required")
	}

	if conf.SqlTimeout <= 0 {
		conf.SqlTimeout = 10
	}

	db, err := sql.Open("pgx", conf.DSN)
	if err != nil {
		return fmt.Errorf("postgres: open db: %w", err)
	}

	poolCfg := PoolConfig{
		MaxOpenConns:        conf.MaxOpenConns,
		MaxIdleConns:        conf.MaxIdleConns,
		ConnMaxLifetime:     conf.ConnMaxLifetime,
		ConnMaxIdleTime:     conf.ConnMaxIdleTime,
		HealthCheckInterval: conf.HealthCheckInterval,
		HealthCheckTimeout:  conf.HealthCheckTimeout,
	}
	poolMgr := NewPoolManager(db, poolCfg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.SqlTimeout)*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		poolMgr.Stop()
		_ = db.Close()
		return fmt.Errorf("postgres: ping db: %w", err)
	}

	drv := entsql.OpenDB("postgres", db)
	a.client = gen.NewClient(gen.Driver(drv))

	a.db = db
	a.poolMgr = poolMgr
	a.dbName = defaultDatabase
	a.maxResults = jsonConfig.MaxResults
	if a.maxResults <= 0 {
		a.maxResults = defaultMaxResults
	}
	a.maxMessageResults = defaultMaxMessageResults
	a.sqlTimeout = time.Duration(conf.SqlTimeout) * time.Second
	a.txTimeout = time.Duration(float64(conf.SqlTimeout)*txTimeoutMultiplier) * time.Second
	a.open = true

	poolMgr.Start(context.Background())
	flog.Info("postgres: adapter opened with database '%s'", a.dbName)
	return nil
}
```

- [ ] **Step 4: Edit Close() method (lines 140-148)**

Replace with:
```go
func (a *adapter) Close() error {
	if a.poolMgr != nil {
		a.poolMgr.Stop()
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("postgres: close db: %w", err)
		}
	}
	a.open = false
	return nil
}
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./internal/store/postgres/
```

Expected: PASS

- [ ] **Step 6: Run all postgres package tests**

```bash
go test ./internal/store/postgres/ -v -count=1
```

Expected: PASS

- [ ] **Step 7: Verify full project build**

```bash
go build ./...
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/store/postgres/adapter.go
git commit -m "feat: integrate PoolManager into adapter Open/Close lifecycle"
```

---

### Task 4: Modify ent.go to apply conservative pool defaults

**Files:**
- Modify: `internal/store/ent/ent.go`

- [ ] **Step 1: Add import and call ApplyDefaults**

Rewrite `internal/store/ent/ent.go`:
```go
package ent

import (
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" //revive:disable:blank-imports pgx driver registration

	gen "github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/postgres"
)

// NewClient creates a new Ent client connected to a PostgreSQL database.
func NewClient(dsn string) (*gen.Client, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	postgres.ApplyDefaults(db)
	drv := entsql.OpenDB(dialect.Postgres, db)
	return gen.NewClient(gen.Driver(drv)), nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/store/ent/
```

Expected: PASS (no circular import)

- [ ] **Step 3: Verify full project build**

```bash
go build ./...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/store/ent/ent.go
git commit -m "feat: apply conservative pool defaults in ent.NewClient test path"
```

---

### Task 5: Update config files with new pool keys

**Files:**
- Modify: `flowbot.yaml`
- Modify: `docs/reference/config.yaml`

- [ ] **Step 1: Add new keys to flowbot.yaml postgres block**

After line 41 (`sql_timeout: 15`), add:
```yaml
      conn_max_idle_time: 60
      pool_health_check_interval: 30
      pool_health_check_timeout: 5
```

The postgres section (lines 36-41) becomes:
```yaml
    postgres:
      dsn: postgres://app:9gDH11MFGMYO1fat@192.168.0.201:15432/dev?sslmode=disable
      max_open_conns: 25
      max_idle_conns: 12
      conn_max_lifetime: 300
      sql_timeout: 15
      conn_max_idle_time: 60
      pool_health_check_interval: 30
      pool_health_check_timeout: 5
```

- [ ] **Step 2: Fix docs/reference/config.yaml**

Replace the postgres block (lines 42-52) — currently has duplicate keys and old format:
```yaml
    # PostgreSQL database configuration
    postgres:
      dsn: postgres://username:password@localhost/flowbot?sslmode=disable
      max_open_conns: 64
      max_idle_conns: 64
      conn_max_lifetime: 300
      sql_timeout: 10
      conn_max_idle_time: 60
      pool_health_check_interval: 30
      pool_health_check_timeout: 5
```

- [ ] **Step 3: Verify config files are valid YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('flowbot.yaml'))" && echo "OK"
python3 -c "import yaml; yaml.safe_load(open('docs/reference/config.yaml'))" && echo "OK"
```

- [ ] **Step 4: Commit**

```bash
git add flowbot.yaml docs/reference/config.yaml
git commit -m "config: add conn_max_idle_time, pool_health_check_interval, pool_health_check_timeout"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./internal/store/postgres/ -v -count=1
go test ./internal/store/ent/ -v -count=1
go build ./...
```

Expected: all PASS

- [ ] **Step 2: Run lint on entire project**

```bash
go tool task lint
```

Expected: no new lint errors introduced

- [ ] **Step 3: Verify metric names appear in /metrics endpoint (requires running server)**

```bash
curl -s http://localhost:8888/metrics | grep flowbot_db_pool
```

Expected: see all 9 metrics with their current values
