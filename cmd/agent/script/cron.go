package script

import (
	"errors"
	"time"

	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func (e *Engine) addCronJob(r Rule) (int, error) {
	schedule, err := cronInterval(r.When)
	if err != nil {
		return 0, errors.New("invalid cron schedule")
	}
	periodicJobHandle, err := e.client.PeriodicJobs().AddSafely(
		river.NewPeriodicJob(
			schedule,
			func() (river.JobArgs, *river.InsertOpts) {
				return r, nil
			},
			nil,
		),
	)
	if err != nil {
		return 0, err
	}
	e.cronJobs.Store(r.Id, int(periodicJobHandle))
	flog.Info("[script] add cron job %+v", periodicJobHandle)
	return int(periodicJobHandle), nil
}

func (e *Engine) removeCronJob(r Rule) {
	cronId, ok := e.cronJobs.Load(r.Id)
	if !ok {
		return
	}
	periodicJobHandle := rivertype.PeriodicJobHandle(cronId.(int))
	e.client.PeriodicJobs().Remove(rivertype.PeriodicJobHandle(periodicJobHandle))
	e.cronJobs.Delete(r.Id)
	flog.Info("[script] remove cron job %+v", periodicJobHandle)
}

type cronIntervalSchedule struct {
	schedule cron.Schedule
}

func cronInterval(when string) (*cronIntervalSchedule, error) {
	p := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	s, err := p.Parse(when)
	if err != nil {
		return nil, err
	}

	return &cronIntervalSchedule{schedule: s}, nil
}

func (s *cronIntervalSchedule) Next(t time.Time) time.Time {
	return s.schedule.Next(t)
}
