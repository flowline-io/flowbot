package bot

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

var clipboard = []Executor{
	{
		Flag: "clipboard_share",
		Run: func(data types.KV) error {
			txt, _ := data.String("txt")
			if txt != "" {
				// app.SendNotification(fyne.NewNotification("clipboard", "share text from chat"))
				// window.Clipboard().SetContent(txt)
				flog.Info("todo")
			}
			return nil
		},
	},
}
