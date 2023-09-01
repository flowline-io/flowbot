package cache

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/flog"
	"os"

	"github.com/redis/go-redis/v9"
)

var DB *redis.Client

func InitCache() {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	if addr == "" || password == "" {
		panic("redis config error")
	}
	DB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
	s := DB.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		panic("redis server error " + err.Error())
	}
}

func Shutdown() {
	err := DB.Close()
	if err != nil {
		flog.Error(err)
		return
	}
	flog.Info("cache stopped")
}
