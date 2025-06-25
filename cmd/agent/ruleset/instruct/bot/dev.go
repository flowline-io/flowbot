package bot

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

func RegisterDev() {
	types.InstructRegister("dev", dev)
}

var dev = []types.Executor{
	{
		Flag: "dev_example",
		Run: func(data types.KV) error {
			flog.Info("dev instruct example %s %s", data, time.Now())
			return nil
		},
	},
}
