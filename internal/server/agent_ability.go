package server

import (
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
)

func initAgentAbility() error {
	abilityagent.SetRunner(chatagent.PipelineAgentRunner{})
	return abilityagent.Register()
}
