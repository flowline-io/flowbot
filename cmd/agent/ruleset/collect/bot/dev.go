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
	_, err := client.Collect(types.FlowkitData{
		//Id:      ImportAgentId,
		Version: types.ApiVersion,
		Content: types.KV{
			"time": time.Now().String(),
		},
	})
	if err != nil {
		flog.Error(err)
	}
}
