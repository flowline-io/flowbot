package agent

import (
	"github.com/allegro/bigcache/v3"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/agent/bot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/robfig/cron/v3"
)

type agentJob struct {
	cache *bigcache.BigCache
}

func (j *agentJob) RunAnki(c *cron.Cron) {
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[anki] stats")
		bot.AnkiStats()
	})
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[anki] review")
		bot.AnkiReview()
	})
}

func (j *agentJob) RunDev(c *cron.Cron) {
	MustAddFunc(c, "0 * * * * *", func() {
		flog.Info("[dev] import")
		bot.DevImport()
	})
}

// MustAddFunc will panic
func MustAddFunc(c *cron.Cron, spec string, cmd func()) {
	_, err := c.AddFunc(spec, cmd)
	if err != nil {
		flog.Panic(err.Error())
	}
}
