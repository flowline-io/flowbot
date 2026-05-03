package rdb

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

var Client *redis.Client

func NewClient(lc fx.Lifecycle, _ *config.Type) (*redis.Client, error) {
	addr := net.JoinHostPort(config.App.Redis.Host, strconv.Itoa(config.App.Redis.Port))
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return nil, fmt.Errorf("redis config error")
	}
	Client = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           config.App.Redis.DB,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	if err := redisotel.InstrumentTracing(Client); err != nil {
		return nil, fmt.Errorf("failed to instrument redis with tracing: %w", err)
	}
	s := Client.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		return nil, fmt.Errorf("redis server error %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			Shutdown(ctx)
			return nil
		},
	})

	return Client, nil
}

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
