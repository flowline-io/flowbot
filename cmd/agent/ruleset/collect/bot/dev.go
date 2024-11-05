package bot

import (
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	ImportCollectId = "import_collect"
)

func DevImport() {
	err := client.Collect(types.CollectData{
		Id: ImportCollectId,
		Content: types.KV{
			"time": time.Now().String(),
		},
	})
	if err != nil {
		flog.Error(err)
	}
}