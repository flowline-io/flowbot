package event

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/redis/go-redis/v9"
	"time"
)

func newRedisClient() (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return nil, fmt.Errorf("redis config error")
	}
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           config.App.Redis.DB,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	s := client.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		return nil, fmt.Errorf("redis server error %w", err)
	}

	return client, nil
}
