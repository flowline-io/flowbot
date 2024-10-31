package bot

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"time"
)

var dev = []Executor{
	{
		Flag: "dev_example",
		Run: func(app any, window any, data types.KV) error {
			flog.Info("dev example %s %s", data, time.Now())
			return nil
		},
	},
}
