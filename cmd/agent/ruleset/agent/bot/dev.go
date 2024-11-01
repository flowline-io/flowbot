package bot

import (
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"time"
)

const (
	ImportAgentId = "import_agent"
)

func DevImport() {
	_, err := client.Agent(types.FlowkitData{
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
