// Package rdb provides a singleton Redis client with connection pool configuration.
package rdb

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Client is the singleton Redis client, provided by NewClient via fx.
var Client *redis.Client

// NewClient creates and returns a single Redis client configured from config.App.Redis.
// Connection pool parameters use go-redis defaults when set to zero, except ReadTimeout
// and WriteTimeout which fall back to 60s for backward compatibility.
func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	addr := net.JoinHostPort(config.App.Redis.Host, strconv.Itoa(config.App.Redis.Port))
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return nil, fmt.Errorf("redis config error")
	}
	Client = redis.NewClient(redisOptions(config.App.Redis))
	if err := redisotel.InstrumentTracing(Client); err != nil {
		return nil, fmt.Errorf("failed to instrument redis with tracing: %w", err)
	}
	s := Client.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		return nil, fmt.Errorf("redis server error %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			Shutdown(ctx)
			return nil
		},
	})

	return Client, nil
}

// redisOptions builds a go-redis Options from the config, applying fallback defaults
// for ReadTimeout and WriteTimeout when they are zero.
func redisOptions(cfg config.Redis) *redis.Options {
	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 60 * time.Second
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 60 * time.Second
	}

	return &redis.Options{
		Addr:            net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		PoolFIFO:        cfg.PoolFIFO,
	}
}

// Shutdown gracefully closes the Redis client with a 5-second timeout.
func Shutdown(ctx context.Context) {
	if Client == nil {
		flog.Warn("redis not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	if err == nil {
		err = Client.Close()
		if err != nil {
			flog.Error(fmt.Errorf("failed to close redis connection: %w", err))
			return
		}
		flog.Info("redis stopped")
	} else {
		flog.Warn("redis connection already lost: %v", err)
	}
}
