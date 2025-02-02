package collect

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/robfig/cron/v3"
)

func Cron() {
	// collect job
	c := cron.New(cron.WithSeconds())
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(24*time.Hour))
	if err != nil {
		flog.Panic(err.Error())
	}

	job := &collectJob{cache: cache}
	job.RunAnki(c)
	job.RunDev(c)
	c.Start()
}
