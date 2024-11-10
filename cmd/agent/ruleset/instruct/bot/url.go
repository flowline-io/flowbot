package bot

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

var url = []Executor{
	{
		Flag: "url_open",
		Run: func(data types.KV) error {
			txt, _ := data.String("url")
			if txt != "" {
				// u, err := netUrl.Parse(txt)
				// if err != nil {
				// 	return err
				// }
				// err = app.OpenURL(u)
				// if err != nil {
				// 	return err
				// }
				// app.SendNotification(fyne.NewNotification("url", "open url"))
				flog.Info("todo")
			}
			return nil
		},
	},
}
