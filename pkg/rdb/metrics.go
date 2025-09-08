package rdb

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func SetMetricsInt64(key string, value int64) {
	Client.Set(context.Background(), metricsKey(key), value, 0)
}

func GetMetricsInt64(key string) int64 {
	r, err := Client.Get(context.Background(), metricsKey(key)).Int64()
	if err != nil {
		flog.Error(err)
	}
	return r
}

func metricsKey(key string) string {
	return fmt.Sprintf("metrics:%s", key)
}
