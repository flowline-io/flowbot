package instruct

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/robfig/cron/v3"
	"time"
)

func Cron() {
	//if preferences.AppConfig().AccessToken == "" {
	//	return
	//}
	// instruct job
	c := cron.New(cron.WithSeconds())
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(time.Hour))
	if err != nil {
		flog.Panic(err.Error())
	}
	// preferences.AppConfig().AccessToken
	token := "" // todo
	job := &instructJob{client: client.NewFlowbot(token), cache: cache}
	_, err = c.AddJob("*/10 * * * * *", job)
	if err != nil {
		flog.Panic(err.Error())
	}
	c.Start()
}
