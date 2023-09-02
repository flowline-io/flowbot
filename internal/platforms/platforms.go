package platforms

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
)

// Driver Functional implementation of the client/server responsible for receiving
// and sending messages (usually HTTP communication)
type Driver interface {
	// HandleMessage handle incoming message
	HandleMessage() (types.MsgPayload, error)
	// HandleEvent handle event
	HandleEvent() (types.EventPayload, error)
}

// Adapter Responsible for converting platform messages to chatbot event/message formats.
type Adapter interface {
	MessageConvert(data any) protocol.Message
	EventConvert(data any) protocol.Event
}

// Action An interface for the application to actively obtain information about the Chatbot implementation or
// robot platform and to control the behavior of the Chatbot implementation or robot.
type Action interface {
	// SendMessage send message
	SendMessage(req protocol.ActionRequest) protocol.ActionResponse
	// RegisterChannels register channels
	RegisterChannels(req protocol.ActionRequest) protocol.ActionResponse
	// RegisterSlashCommands register slash commands
	RegisterSlashCommands(req protocol.ActionRequest) protocol.ActionResponse
}
