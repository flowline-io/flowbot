package rdb

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

var Client *redis.Client

func NewClient(lc fx.Lifecycle, _ config.Type) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
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
			return Client.Close()
		},
	})

	return Client, nil
}

func Shutdown() {
	if Client == nil {
		flog.Warn("redis not initialized")
		return
	}

	_, err := Client.Ping(context.Background()).Result()
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

func SetInt64(key string, value int64) {
	Client.Set(context.Background(), key, value, 0)
}

func GetInt64(key string) int64 {
	r, err := Client.Get(context.Background(), key).Int64()
	if err != nil {
		flog.Error(err)
	}
	return r
}
