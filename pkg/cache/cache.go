package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/redis/go-redis/v9"
)

var DB *redis.Client

func InitCache() {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		panic("redis config error")
	}
	DB = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           config.App.Redis.DB,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	s := DB.Ping(context.Background())
	_, err := s.Result()
	if err != nil {
		panic("redis server error " + err.Error())
	}
}

func Shutdown() {
	_, err := DB.Ping(context.Background()).Result()
	if err == nil {
		err = DB.Close()
		if err != nil {
			flog.Error(err)
			return
		}
	}
	flog.Info("cache stopped")
}

func SetInt64(key string, value int64) {
	DB.Set(context.Background(), key, value, 0)
}

func GetInt64(key string) int64 {
	r, err := DB.Get(context.Background(), key).Int64()
	if err != nil {
		flog.Error(err)
	}
	return r
}
