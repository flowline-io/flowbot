package download

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

var cronRules = []cron.Rule{
	{
		Name: "download_clean_expired_files",
		When: "* * * * *",
		Action: func(types.Context) []types.MsgPayload {
			downloadPath := config.App.Flowbot.DownloadPath
			if !utils.FileExist(downloadPath) {
				return nil
			}

			err := filepath.Walk(downloadPath, func(path string, info fs.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}
				// expired file
				if info.ModTime().Before(time.Now().Add(-24 * time.Hour)) {
					return os.Remove(path)
				}
				return nil
			})
			if err != nil {
				flog.Error(err)
			}

			return nil
		},
	},
}
