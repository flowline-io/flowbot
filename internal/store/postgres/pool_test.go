package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/prometheus/client_golang/prometheus"
)

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
				HealthCheckInterval: 0,
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

func TestPoolConfig_FromJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected PoolConfig
	}{
		{
			name: "all fields populated",
			json: `{"max_open_conns":50,"max_idle_conns":20,"conn_max_lifetime":600,"conn_max_idle_time":120,"pool_health_check_interval":60,"pool_health_check_timeout":10}`,
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
			name:     "empty json uses zero values",
			json:     `{}`,
			expected: PoolConfig{},
		},
		{
			name: "partial json fills only present keys",
			json: `{"max_open_conns":100,"conn_max_idle_time":90}`,
			expected: PoolConfig{
				MaxOpenConns:    100,
				ConnMaxIdleTime: 90,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc PoolConfig
			if err := sonic.Unmarshal([]byte(tt.json), &pc); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if pc.MaxOpenConns != tt.expected.MaxOpenConns {
				t.Errorf("MaxOpenConns: got %d, want %d", pc.MaxOpenConns, tt.expected.MaxOpenConns)
			}
			if pc.MaxIdleConns != tt.expected.MaxIdleConns {
				t.Errorf("MaxIdleConns: got %d, want %d", pc.MaxIdleConns, tt.expected.MaxIdleConns)
			}
			if pc.ConnMaxLifetime != tt.expected.ConnMaxLifetime {
				t.Errorf("ConnMaxLifetime: got %d, want %d", pc.ConnMaxLifetime, tt.expected.ConnMaxLifetime)
			}
			if pc.ConnMaxIdleTime != tt.expected.ConnMaxIdleTime {
				t.Errorf("ConnMaxIdleTime: got %d, want %d", pc.ConnMaxIdleTime, tt.expected.ConnMaxIdleTime)
			}
			if pc.HealthCheckInterval != tt.expected.HealthCheckInterval {
				t.Errorf("HealthCheckInterval: got %d, want %d", pc.HealthCheckInterval, tt.expected.HealthCheckInterval)
			}
			if pc.HealthCheckTimeout != tt.expected.HealthCheckTimeout {
				t.Errorf("HealthCheckTimeout: got %d, want %d", pc.HealthCheckTimeout, tt.expected.HealthCheckTimeout)
			}
		})
	}
}

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

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "nil db is safe",
			fn: func(_ *testing.T) {
				ApplyDefaults(nil)
			},
		},
		{
			name: "sets expected conservative limits",
			fn: func(t *testing.T) {
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
			},
		},
		{
			name: "can set different values later",
			fn: func(t *testing.T) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestRegisterMetrics_Idempotent(t *testing.T) {
	t.Run("returns same instance on repeated calls", func(t *testing.T) {
		m1 := registerMetrics()
		m2 := registerMetrics()
		if m1 != m2 {
			t.Error("registerMetrics should return the same instance on repeated calls")
		}
	})
}

func TestRegisterMetrics_AllNames(t *testing.T) {
	t.Run("registers all expected metric names", func(t *testing.T) {
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
	})
}

func TestPoolManager_StartStop(t *testing.T) {
	t.Run("start and stop lifetime", func(t *testing.T) {
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

		ctx := t.Context()

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
	})
}

func TestPoolManager_StopTwice(t *testing.T) {
	t.Run("double stop is safe", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		pm := NewPoolManager(db, PoolConfig{
			HealthCheckInterval: 1,
			HealthCheckTimeout:  1,
		})

		ctx := t.Context()

		pm.Start(ctx)
		pm.Stop()
		pm.Stop()
	})
}

func TestPoolManager_StopUnstarted(t *testing.T) {
	t.Run("stop without start is safe", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		pm := NewPoolManager(db, PoolConfig{})
		pm.Stop()
	})
}

func TestPoolManager_PingerDisabled(t *testing.T) {
	t.Run("zero health check interval disables pinger", func(t *testing.T) {
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
	})
}

func TestPoolManager_HealthCheckIncrementsCounters(t *testing.T) {
	t.Run("health check increments total counter", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		pm := NewPoolManager(db, PoolConfig{
			HealthCheckInterval: 0,
			HealthCheckTimeout:  1,
		})

		pm.healthCheck(context.Background())

		metrics, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatalf("gather metrics failed: %v", err)
		}
		var found bool
		for _, mf := range metrics {
			if mf.GetName() == "flowbot_db_pool_health_check_total" {
				if len(mf.Metric) > 0 && mf.Metric[0].Counter != nil {
					if mf.Metric[0].Counter.GetValue() >= 1 {
						found = true
					}
				}
			}
		}
		if !found {
			t.Error("healthTotal counter should be >= 1 after health check")
		}
	})
}

func TestPoolManager_CollectStats(t *testing.T) {
	t.Run("collect stats populates gauges", func(t *testing.T) {
		db, err := sql.Open("pgx", "postgres://localhost:5432/nonexistent?sslmode=disable&connect_timeout=1")
		if err != nil {
			t.Skipf("skipping: cannot open test db: %v", err)
		}
		defer db.Close()

		pm := NewPoolManager(db, PoolConfig{})
		pm.collectStats()

		metrics, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatalf("gather metrics failed: %v", err)
		}

		expectedMetrics := map[string]bool{
			"flowbot_db_pool_connections_open":            false,
			"flowbot_db_pool_connections_idle":            false,
			"flowbot_db_pool_connections_in_use":          false,
			"flowbot_db_pool_wait_count_total":            false,
			"flowbot_db_pool_wait_duration_seconds_total": false,
			"flowbot_db_pool_max_idle_closed_total":       false,
			"flowbot_db_pool_max_lifetime_closed_total":   false,
		}

		for _, mf := range metrics {
			if _, ok := expectedMetrics[mf.GetName()]; ok {
				if len(mf.Metric) > 0 && mf.Metric[0].Gauge != nil {
					expectedMetrics[mf.GetName()] = true
				}
			}
		}

		for name, found := range expectedMetrics {
			if !found {
				t.Errorf("gauge metric %q not found or has no value", name)
			}
		}
	})
}
