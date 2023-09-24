package protocol

import "fmt"

type ResponseStatus string

const (
	Success ResponseStatus = "ok"
	Failed  ResponseStatus = "failed"

	SuccessCode = int64(0)
)

const (
	GetLatestEventsAction     = "get_latest_events"
	GetSupportedActionsAction = "get_supported_actions"
	GetStatusAction           = "get_status"
	GetVersionAction          = "get_version"

	SendMessageAction = "send_message"

	GetUserInfoAction = "get_user_info"

	CreateChannelAction  = "create_channel"
	GetChannelInfoAction = "get_channel_info"
	GetChannelListAction = "get_channel_list"
)

type Request struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

type Response struct {
	// Execution status (success or failure), must be one of ok and failed,
	// indicating successful and unsuccessful execution, respectively.
	Status ResponseStatus `json:"status"`
	// The return code, which must conform to the return code rules defined later on this page
	RetCode int64 `json:"retcode,omitempty"`
	// Response data
	Data any `json:"data"`
	// Error message, it is recommended to fill in a human-readable error message when the action fails to execute,
	// or an empty string when it succeeds.
	Message string `json:"message,omitempty"`
}

type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func NewError(code int64, message string) *Error {
	return &Error{Code: code, Message: message}
}

func (e Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func (e Error) GetCode() int64 {
	return e.Code
}

func (e Error) GetMessage() string {
	return e.Message
}

// Request Error (10xxx)

// ErrBadRequest  Formatting errors (including implementations that do not support MessagePack),
// missing required fields, or incorrect field types
var ErrBadRequest = NewError(10001, "bad request")

// ErrUnsupportedAction The Chatbot implementation does not implement this action
var ErrUnsupportedAction = NewError(10002, "unsupported action")

// ErrBadParam Missing parameter or wrong parameter type
var ErrBadParam = NewError(10003, "bad parameter")

// ErrUnsupported The Chatbot implementation does not implement the semantics of this parameter
var ErrUnsupported = NewError(10004, "unsupported")

// ErrUnsupportedSegment The Chatbot implementation does not implement this segment type.
var ErrUnsupportedSegment = NewError(10005, "unsupported segment")

// ErrBadSegmentType Missing parameter or wrong parameter type
var ErrBadSegmentType = NewError(10006, "bad segment type")

// ErrBadSegmentData The Chatbot implementation does not implement the semantics of this parameter
var ErrBadSegmentData = NewError(10007, "bad segment data")

// ErrMethodNotAllowed Invalid HTTP method
var ErrMethodNotAllowed = NewError(10008, "invalid http method")

// ErrWhoAmI Chatbot implements support for multiple bot accounts on a single Chatbot Connect connection,
// but the action request does not specify the account to be used
var ErrWhoAmI = NewError(10101, "who am i")

// ErrUnknownSelf The bot account specified by the action request does not exist
var ErrUnknownSelf = NewError(10102, "unknown self")

// Handler Error (20xxx)

// ErrBadHandler Response status not set correctly, etc.
var ErrBadHandler = NewError(20001, "bad handler")

// ErrInternalHandler An uncaught and unexpected exception has occurred within the Chatbot implementation.
var ErrInternalHandler = NewError(20002, "internal handler")

// Execution Error (30xxx)

// ErrDatabaseError Such as database query failure
var ErrDatabaseError = NewError(31001, "database error")

// ErrDatabaseReadError Such as database read failure
var ErrDatabaseReadError = NewError(31002, "database read error")

// ErrDatabaseWriteError Such as database write failure
var ErrDatabaseWriteError = NewError(31003, "database write error")

// ErrFilesystemError If reading or writing a file fails, etc.
var ErrFilesystemError = NewError(32001, "filesystem error")

// ErrNetworkError e.g. failed to download a file, etc.
var ErrNetworkError = NewError(33001, "network error")

// ErrPlatformError e.g. failure to send messages due to bot platform limitations, etc.
var ErrPlatformError = NewError(34001, "platform error")

// ErrLoginError Such as trying to send a message to a non-existent user
var ErrLoginError = NewError(35001, "login error")

// ErrIAmTired A Chatbot realizes the decision to strike.
var ErrIAmTired = NewError(36001, "i am tired")

// Retention Error (40xxx to 50xxx)

// Business error (60xxx to 90xxx)

// ErrTokenError missing, invalid or expired access token
var ErrTokenError = NewError(60001, "missing, invalid or expired access token")

func NewSuccessResponse(data any) Response {
	return Response{
		Status:  Success,
		RetCode: SuccessCode,
		Data:    data,
	}
}

func NewFailedResponse(e *Error) Response {
	return Response{
		Status:  Failed,
		RetCode: e.GetCode(),
		Message: e.GetMessage(),
	}
}
