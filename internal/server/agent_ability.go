package server

import (
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/capability/core"
)

func initAgentAbility() error {
	core.SetRunner(chatagent.PipelineAgentRunner{})
	return nil
}
