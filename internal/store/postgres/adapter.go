// Package postgres implements the PostgreSQL storage adapter.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/bytedance/sonic"
	_ "github.com/jackc/pgx/v5/stdlib" //revive:disable:blank-imports pgx driver registration
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	adapterName = "postgres"

	defaultDatabase = "flowbot"

	defaultMaxResults        = 1024
	defaultMaxMessageResults = 100

	txTimeoutMultiplier = 1.5
)

type configType struct {
	DSN                 string `json:"dsn,omitempty"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime     int    `json:"conn_max_lifetime,omitempty"`
	ConnMaxIdleTime     int    `json:"conn_max_idle_time,omitempty"`
	SqlTimeout          int    `json:"sql_timeout,omitempty"`
	HealthCheckInterval int    `json:"pool_health_check_interval,omitempty"`
	HealthCheckTimeout  int    `json:"pool_health_check_timeout,omitempty"`
}

// Init registers the postgres adapter with the store layer.
func Init() {
	store.RegisterAdapter(&adapter{})
}

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

	if conf.SqlTimeout <= 0 {
		conf.SqlTimeout = 10
	}

	db, err := otelsql.Open("pgx", conf.DSN,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
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
	store.SetQueryLimits(a.maxResults, a.maxMessageResults)
	a.sqlTimeout = time.Duration(conf.SqlTimeout) * time.Second
	a.txTimeout = time.Duration(float64(conf.SqlTimeout)*txTimeoutMultiplier) * time.Second
	a.open = true

	poolMgr.Start(context.Background())
	flog.Info("postgres: adapter opened with database '%s'", a.dbName)
	return nil
}

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

func (a *adapter) IsOpen() bool {
	return a.open
}

func (*adapter) GetName() string {
	return adapterName
}

func (a *adapter) Stats() any {
	if a.db != nil {
		return a.db.Stats()
	}
	return nil
}

func (a *adapter) GetClient() *gen.Client {
	return a.client
}

// Ping checks PostgreSQL connectivity and returns the round-trip latency.
func (a *adapter) Ping(ctx context.Context) (time.Duration, error) {
	if a.db == nil {
		return 0, fmt.Errorf("postgres: database not initialized")
	}
	start := time.Now()
	err := a.db.PingContext(ctx)
	return time.Since(start), err
}

func (a *adapter) GetDB() any {
	return a.client
}
