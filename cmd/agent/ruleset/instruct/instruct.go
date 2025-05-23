package instruct

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/robfig/cron/v3"
)

func Cron() {
	// instruct job
	c := cron.New(cron.WithSeconds())
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(time.Hour))
	if err != nil {
		flog.Panic(err.Error())
	}

	job := &instructJob{cache: cache}
	_, err = c.AddJob("*/10 * * * * *", job)
	if err != nil {
		flog.Panic(err.Error())
	}
	c.Start()
}
