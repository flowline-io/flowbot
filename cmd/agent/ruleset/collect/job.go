package collect

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/collect/bot"
	"github.com/flowline-io/flowbot/pkg/flog"
)

type collectJob struct{}

func (j *collectJob) Run(c *cron.Cron) {
	// anki
	MustAddFunc(c, "0 * * * * *", func(ctx context.Context) error {
		flog.Info("[anki] stats")
		bot.AnkiStats()
		return nil
	})
	MustAddFunc(c, "0 * * * * *", func(ctx context.Context) error {
		flog.Info("[anki] review")
		bot.AnkiReview()
		return nil
	})

	// dev
	MustAddFunc(c, "0 * * * * *", func(ctx context.Context) error {
		flog.Info("[dev] import")
		bot.DevImport()
		return nil
	})
}

// MustAddFunc will panic
func MustAddFunc(c *cron.Cron, spec string, cmd func(ctx context.Context) error) {
	_, err := c.AddFunc(spec, cmd)
	if err != nil {
		flog.Panic("collect cron add job failed %v", err)
	}
}
