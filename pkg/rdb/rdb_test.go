package rdb

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestRedisOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 config.Redis
		wantAddr            string
		wantDB              int
		wantPool            int
		wantMinIdle         int
		wantReadTO          time.Duration
		wantWriteTO         time.Duration
		wantRetries         int
		wantFIFO            bool
		wantMinRetryBackoff time.Duration
		wantMaxRetryBackoff time.Duration
		wantDialTimeout     time.Duration
		wantPoolTimeout     time.Duration
		wantConnMaxIdleTime time.Duration
		wantConnMaxLifetime time.Duration
	}{
		{
			name: "all pool fields set explicitly",
			cfg: config.Redis{
				Host:            "redis.example.com",
				Port:            6379,
				DB:              2,
				Password:        "secret",
				PoolSize:        20,
				MinIdleConns:    5,
				MaxRetries:      5,
				MinRetryBackoff: 100 * time.Millisecond,
				MaxRetryBackoff: 5 * time.Second,
				DialTimeout:     3 * time.Second,
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				PoolTimeout:     10 * time.Second,
				ConnMaxIdleTime: 10 * time.Minute,
				ConnMaxLifetime: 1 * time.Hour,
				PoolFIFO:        true,
			},
			wantAddr:            "redis.example.com:6379",
			wantDB:              2,
			wantPool:            20,
			wantMinIdle:         5,
			wantReadTO:          30 * time.Second,
			wantWriteTO:         30 * time.Second,
			wantRetries:         5,
			wantFIFO:            true,
			wantMinRetryBackoff: 100 * time.Millisecond,
			wantMaxRetryBackoff: 5 * time.Second,
			wantDialTimeout:     3 * time.Second,
			wantPoolTimeout:     10 * time.Second,
			wantConnMaxIdleTime: 10 * time.Minute,
			wantConnMaxLifetime: 1 * time.Hour,
		},
		{
			name: "zero pool config uses go-redis defaults",
			cfg: config.Redis{
				Host:     "localhost",
				Port:     6379,
				DB:       0,
				Password: "",
			},
			wantAddr:            "localhost:6379",
			wantDB:              0,
			wantPool:            0,
			wantMinIdle:         0,
			wantReadTO:          60 * time.Second,
			wantWriteTO:         60 * time.Second,
			wantRetries:         0,
			wantFIFO:            false,
			wantMinRetryBackoff: 0,
			wantMaxRetryBackoff: 0,
			wantDialTimeout:     0,
			wantPoolTimeout:     0,
			wantConnMaxIdleTime: 0,
			wantConnMaxLifetime: 0,
		},
		{
			name: "zero read and write timeout falls back to 60s",
			cfg: config.Redis{
				Host:         "127.0.0.1",
				Port:         6380,
				DB:           1,
				Password:     "pwd",
				ReadTimeout:  0,
				WriteTimeout: 0,
			},
			wantAddr:            "127.0.0.1:6380",
			wantDB:              1,
			wantPool:            0,
			wantMinIdle:         0,
			wantReadTO:          60 * time.Second,
			wantWriteTO:         60 * time.Second,
			wantRetries:         0,
			wantFIFO:            false,
			wantMinRetryBackoff: 0,
			wantMaxRetryBackoff: 0,
			wantDialTimeout:     0,
			wantPoolTimeout:     0,
			wantConnMaxIdleTime: 0,
			wantConnMaxLifetime: 0,
		},
		{
			name: "partial pool config with subset of fields",
			cfg: config.Redis{
				Host:            "10.0.0.1",
				Port:            6379,
				DB:              3,
				Password:        "pass",
				PoolSize:        50,
				MinIdleConns:    10,
				ConnMaxLifetime: 30 * time.Minute,
			},
			wantAddr:            "10.0.0.1:6379",
			wantDB:              3,
			wantPool:            50,
			wantMinIdle:         10,
			wantReadTO:          60 * time.Second,
			wantWriteTO:         60 * time.Second,
			wantRetries:         0,
			wantFIFO:            false,
			wantMinRetryBackoff: 0,
			wantMaxRetryBackoff: 0,
			wantDialTimeout:     0,
			wantPoolTimeout:     0,
			wantConnMaxIdleTime: 0,
			wantConnMaxLifetime: 30 * time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := redisOptions(tt.cfg)

			assert.Equal(t, tt.wantAddr, opts.Addr)
			assert.Equal(t, tt.wantDB, opts.DB)
			assert.Equal(t, tt.wantPool, opts.PoolSize)
			assert.Equal(t, tt.wantMinIdle, opts.MinIdleConns)
			assert.Equal(t, tt.wantReadTO, opts.ReadTimeout)
			assert.Equal(t, tt.wantWriteTO, opts.WriteTimeout)
			assert.Equal(t, tt.wantRetries, opts.MaxRetries)
			assert.Equal(t, tt.wantFIFO, opts.PoolFIFO)
			assert.Equal(t, tt.wantMinRetryBackoff, opts.MinRetryBackoff)
			assert.Equal(t, tt.wantMaxRetryBackoff, opts.MaxRetryBackoff)
			assert.Equal(t, tt.wantDialTimeout, opts.DialTimeout)
			assert.Equal(t, tt.wantPoolTimeout, opts.PoolTimeout)
			assert.Equal(t, tt.wantConnMaxIdleTime, opts.ConnMaxIdleTime)
			assert.Equal(t, tt.wantConnMaxLifetime, opts.ConnMaxLifetime)
			assert.Equal(t, tt.cfg.Password, opts.Password)
		})
	}
}

func TestShutdown(t *testing.T) {
	t.Run("nil client does not panic", func(t *testing.T) {
		prev := Client
		Client = nil
		t.Cleanup(func() { Client = prev })

		require.NotPanics(t, func() {
			Shutdown(context.Background())
		})
	})

	t.Run("healthy client calls Close", func(t *testing.T) {
		prev := Client
		t.Cleanup(func() { Client = prev })

		mr := miniredis.RunT(t)
		Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})

		require.NotPanics(t, func() {
			Shutdown(context.Background())
		})
		// After Shutdown, Ping should fail since Close was called.
		_, err := Client.Ping(context.Background()).Result()
		require.Error(t, err)
	})

	t.Run("unreachable client still calls Close", func(t *testing.T) {
		prev := Client
		t.Cleanup(func() { Client = prev })

		mr := miniredis.RunT(t)
		Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		mr.Close() // make Redis unreachable

		require.NotPanics(t, func() {
			Shutdown(context.Background())
		})
		// Client.Close was called, so Ping should fail.
		_, err := Client.Ping(context.Background()).Result()
		require.Error(t, err)
	})

	t.Run("already closed client does not panic", func(t *testing.T) {
		prev := Client
		t.Cleanup(func() { Client = prev })

		mr := miniredis.RunT(t)
		Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		require.NoError(t, Client.Close())

		require.NotPanics(t, func() {
			Shutdown(context.Background())
		})
	})
}
