package instruct

import (
	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func Cron() {
	// instruct job
	c := cron.New(cron.WithSeconds())
	ac, err := cache.NewCache(config.Type{})
	if err != nil {
		flog.Panic(err.Error())
	}

	job := &instructJob{cache: ac}
	_, err = c.AddJob("*/10 * * * * *", job)
	if err != nil {
		flog.Panic(err.Error())
	}
	c.Start()
}
