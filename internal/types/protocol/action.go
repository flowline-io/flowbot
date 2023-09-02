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
	Status  ResponseStatus `json:"status"`
	RetCode int64          `json:"retcode"`
	Data    any            `json:"data"`
	Message string         `json:"message"`
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

// Request Error

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

// ErrWhoAmI Chatbot implements support for multiple bot accounts on a single Chatbot Connect connection,
// but the action request does not specify the account to be used
var ErrWhoAmI = NewError(10101, "who am i")

// ErrUnknownSelf The bot account specified by the action request does not exist
var ErrUnknownSelf = NewError(10102, "unknown self")

// Handler Error

// ErrBadHandler Response status not set correctly, etc.
var ErrBadHandler = NewError(20001, "bad handler")

// ErrInternalHandler An uncaught and unexpected exception has occurred within the Chatbot implementation.
var ErrInternalHandler = NewError(20002, "internal handler")

// Execution Error

// ErrDatabaseError Such as database query failure
var ErrDatabaseError = NewError(31001, "database error")

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
