package bot

import (
	"time"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
)

var dev = []Executor{
	{
		Flag: "dev_example",
		Run: func(data types.KV) error {
			flog.Info("dev instruct example %s %s", data, time.Now())
			return nil
		},
	},
}
