package script

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/influxdata/cron"
)

func (e *Engine) cron() {
	// todo cron manager
}

func (e *Engine) startCron() error {
	return nil
}

func (e *Engine) changeCron() error {
	return nil
}

func (e *Engine) cronScheduler(r Rule) {
	flog.Debug("cron script %s scheduler start", r.Id)
	p, err := cron.ParseUTC(r.When)
	if err != nil {
		flog.Error(err)
		return
	}
	nextTime, err := p.Next(time.Now())
	if err != nil {
		flog.Error(err)
		return
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-e.stop:
			flog.Info("cron script %s scheduler stopped", r.Id)
			return
		case <-ticker.C:
			if nextTime.Format("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
				continue
			}

			// push queue todo

			nextTime, err = p.Next(time.Now())
			if err != nil {
				flog.Error(err)
			}
		}
	}
}

type Rule struct {
	Id      string
	When    string
	Path    string
	Timeout time.Duration
}
