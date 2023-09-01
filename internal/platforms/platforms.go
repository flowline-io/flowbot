package platforms

import (
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/channels"
)

type Platform interface {
	// SendMessage send message
	SendMessage(msg types.MsgPayload) error
	// HandleMessage handle incoming message
	HandleMessage() (types.MsgPayload, error)
	// HandleEvent handle event
	HandleEvent() (types.EventPayload, error)
	// RegisterChannels register channels
	RegisterChannels(rules map[string]channels.Publisher) error
	// RegisterSlashCommands register slash commands
	RegisterSlashCommands(rules map[string]command.Ruleset) error
}
