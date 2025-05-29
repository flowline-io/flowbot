package protocol

import (
	"github.com/gofiber/fiber/v3"
)

// Driver Functional implementation of the client/server responsible for receiving
// and sending messages (usually HTTP communication)
type Driver interface {
	// HttpServer The application can actively access the Chatbot implementation.
	HttpServer(ctx fiber.Ctx) error
	// HttpWebhookClient Chatbot implements active access to applications
	HttpWebhookClient(message Message) error
	// WebSocketClient The application can actively access the Chatbot implementation.
	WebSocketClient()
	// WebSocketServer Chatbot implements active access to applications
	WebSocketServer()
	// Shoutdown Shut down the driver
	Shoutdown() error
}

// Adapter Responsible for converting platform messages to chatbot event/message formats.
type Adapter interface {
	MessageConvert(data any) Message
	EventConvert(data any) Event
}

// Action An interface for the application to actively obtain information about the Chatbot implementation or
// robot platform and to control the behavior of the Chatbot implementation or robot.
type Action interface {
	// GetLatestEvents get latest events, Only the HTTP communication method must be supported for polling for events.
	GetLatestEvents(req Request) Response
	// GetSupportedActions get supported actions
	GetSupportedActions(req Request) Response
	// GetStatus get status
	GetStatus(req Request) Response
	// GetVersion get version
	GetVersion(req Request) Response

	// SendMessage send message
	SendMessage(req Request) Response

	// GetUserInfo get user info
	GetUserInfo(req Request) Response

	// CreateChannel create channel
	CreateChannel(req Request) Response
	// GetChannelInfo get channel info
	GetChannelInfo(req Request) Response
	// GetChannelList get channel list
	GetChannelList(req Request) Response

	// RegisterChannels register channels
	RegisterChannels(req Request) Response
	// RegisterSlashCommands register slash commands
	RegisterSlashCommands(req Request) Response
}
