package download

import (
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/utils"
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
			downloadPath := os.Getenv("DOWNLOAD_PATH")
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
				logs.Err.Println(err)
			}

			return nil
		},
	},
}