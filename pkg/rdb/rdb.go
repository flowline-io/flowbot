// Package rdb provides a singleton Redis client with connection pool configuration.
package rdb

import (
	"context"
	"errors"
	"fmt"
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
	opts, err := redisOptions(config.App.Redis)
	if err != nil {
		return nil, fmt.Errorf("redis options: %w", err)
	}
	Client = redis.NewClient(opts)
	if err := redisotel.InstrumentTracing(Client); err != nil {
		return nil, fmt.Errorf("failed to instrument redis with tracing: %w", err)
	}
	s := Client.Ping(context.Background())
	_, err = s.Result()
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

// redisOptions builds go-redis Options from redis.url plus optional pool overrides.
// ReadTimeout and WriteTimeout fall back to 60s when zero.
func redisOptions(cfg config.Redis) (*redis.Options, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("redis.url is empty")
	}
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis.url: %w", err)
	}

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 60 * time.Second
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 60 * time.Second
	}

	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns
	opts.MaxRetries = cfg.MaxRetries
	opts.MinRetryBackoff = cfg.MinRetryBackoff
	opts.MaxRetryBackoff = cfg.MaxRetryBackoff
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = readTimeout
	opts.WriteTimeout = writeTimeout
	opts.PoolTimeout = cfg.PoolTimeout
	opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	opts.ConnMaxLifetime = cfg.ConnMaxLifetime
	opts.PoolFIFO = cfg.PoolFIFO

	return opts, nil
}

// Shutdown gracefully closes the Redis client with a 5-second timeout.
// Client.Close is always called to release the connection pool, even when
// the ping fails (e.g. Redis is unreachable during shutdown).
func Shutdown(ctx context.Context) {
	if Client == nil {
		flog.Warn("redis not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := Client.Ping(ctx).Result(); err != nil {
		if errors.Is(err, redis.ErrClosed) {
			flog.Debug("redis connection already lost: %v", err)
		} else {
			flog.Warn("redis connection already lost: %v", err)
		}
	}

	if err := Client.Close(); err != nil {
		if errors.Is(err, redis.ErrClosed) {
			flog.Debug("redis connection already closed: %v", err)
			return
		}
		flog.Error(fmt.Errorf("failed to close redis connection: %w", err))
		return
	}
	flog.Info("redis stopped")
}
