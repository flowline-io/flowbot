package collect

import (
	"github.com/flc1125/go-cron/v4"
)

func Cron() {
	// collect job
	c := cron.New(cron.WithSeconds())

	job := &collectJob{}
	job.RunAnki(c)
	job.RunDev(c)
	c.Start()
}
