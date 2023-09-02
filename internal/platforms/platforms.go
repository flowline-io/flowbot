package platforms

import (
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/gofiber/fiber/v2"
)

// Driver Functional implementation of the client/server responsible for receiving
// and sending messages (usually HTTP communication)
type Driver interface {
	// HttpServer The application can actively access the Chatbot implementation.
	HttpServer(app *fiber.App) error
	// HttpWebhookClient Chatbot implements active access to applications
	HttpWebhookClient(message protocol.Message) error
	// WebSocketClient The application can actively access the Chatbot implementation.
	WebSocketClient(stop <-chan bool) error
	// WebSocketServer Chatbot implements active access to applications
	WebSocketServer(stop <-chan bool) error
}

// Adapter Responsible for converting platform messages to chatbot event/message formats.
type Adapter interface {
	MessageConvert(data any) protocol.Message
	EventConvert(data any) protocol.Event
}

// Action An interface for the application to actively obtain information about the Chatbot implementation or
// robot platform and to control the behavior of the Chatbot implementation or robot.
type Action interface {
	// GetLatestEvents get latest events, Only the HTTP communication method must be supported for polling for events.
	GetLatestEvents(req protocol.ActionRequest) protocol.ActionResponse
	// GetSupportedActions get supported actions
	GetSupportedActions(req protocol.ActionRequest) protocol.ActionResponse
	// GetStatus get status
	GetStatus(req protocol.ActionRequest) protocol.ActionResponse
	// GetVersion get version
	GetVersion(req protocol.ActionRequest) protocol.ActionResponse

	// SendMessage send message
	SendMessage(req protocol.ActionRequest) protocol.ActionResponse

	// GetUserInfo get user info
	GetUserInfo(req protocol.ActionRequest) protocol.ActionResponse

	// CreateChannel create channel
	CreateChannel(req protocol.ActionRequest) protocol.ActionResponse
	// GetChannelInfo get channel info
	GetChannelInfo(req protocol.ActionRequest) protocol.ActionResponse
	// GetChannelList get channel list
	GetChannelList(req protocol.ActionRequest) protocol.ActionResponse

	// RegisterChannels register channels
	RegisterChannels(req protocol.ActionRequest) protocol.ActionResponse
	// RegisterSlashCommands register slash commands
	RegisterSlashCommands(req protocol.ActionRequest) protocol.ActionResponse
}
