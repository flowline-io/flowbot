package bot

import (
	"github.com/flowline-io/flowbot/cmd/agent/desktop"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

func RegisterClipboard() {
	types.InstructRegister("clipboard", clipboard)
}

var clipboard = []types.Executor{
	{
		Flag: "clipboard_share",
		Run: func(data types.KV) error {
			txt, _ := data.String("txt")
			if txt != "" {
				flog.Info("share clipboard %s", txt)
				d := desktop.Desktop{}
				d.Notify("clipboard", txt)
			}
			return nil
		},
	},
}
