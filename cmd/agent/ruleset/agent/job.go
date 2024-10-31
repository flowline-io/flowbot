package agent

import (
	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/agent/bot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/robfig/cron/v3"
)

type agentJob struct {
	cache  *bigcache.BigCache
	client *client.Flowbot
}

func (j *agentJob) RunAnki(c *cron.Cron) {
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[agent] anki stats")
		bot.AnkiStats(j.client)
	})
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[agent] anki review")
		bot.AnkiReview(j.client)
	})
}

func (j *agentJob) RunDev(c *cron.Cron) {
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[agent] dev import")
		bot.DevImport(j.client)
	})
}

// MustAddFunc will panic
func MustAddFunc(c *cron.Cron, spec string, cmd func()) {
	_, err := c.AddFunc(spec, cmd)
	if err != nil {
		flog.Panic(err.Error())
	}
}
