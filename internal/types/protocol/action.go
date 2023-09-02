package protocol

import "fmt"

type ResponseStatus string

const (
	Success ResponseStatus = "ok"
	Failed  ResponseStatus = "failed"
)

const (
	SuccessCode = int64(0)

	SendMessageAction   = "send_message"
	DeleteMessageAction = "delete_message"
)

type ActionRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

type ActionResponse struct {
	Status  ResponseStatus `json:"status"`
	RetCode int64          `json:"retcode"`
	Data    any            `json:"data"`
	Message string         `json:"message"`
}

type ActionError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func NewActionError(code int64, message string) *ActionError {
	return &ActionError{Code: code, Message: message}
}

func (e ActionError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func (e ActionError) GetCode() int64 {
	return e.Code
}

func (e ActionError) GetMessage() string {
	return e.Message
}

// Request Error

// ErrBadRequest  Formatting errors (including implementations that do not support MessagePack),
// missing required fields, or incorrect field types
var ErrBadRequest = NewActionError(10001, "bad request")

// ErrUnsupportedAction The Chatbot implementation does not implement this action
var ErrUnsupportedAction = NewActionError(10002, "unsupported action")

// ErrBadParam Missing parameter or wrong parameter type
var ErrBadParam = NewActionError(10003, "bad parameter")

// ErrUnsupported The Chatbot implementation does not implement the semantics of this parameter
var ErrUnsupported = NewActionError(10004, "unsupported")

// ErrUnsupportedSegment The Chatbot implementation does not implement this segment type.
var ErrUnsupportedSegment = NewActionError(10005, "unsupported segment")

// ErrBadSegmentType Missing parameter or wrong parameter type
var ErrBadSegmentType = NewActionError(10006, "bad segment type")

// ErrBadSegmentData The Chatbot implementation does not implement the semantics of this parameter
var ErrBadSegmentData = NewActionError(10007, "bad segment data")

// ErrWhoAmI Chatbot implements support for multiple bot accounts on a single Chatbot Connect connection,
// but the action request does not specify the account to be used
var ErrWhoAmI = NewActionError(10101, "who am i")

// ErrUnknownSelf The bot account specified by the action request does not exist
var ErrUnknownSelf = NewActionError(10102, "unknown self")

// Handler Error

// ErrBadHandler Response status not set correctly, etc.
var ErrBadHandler = NewActionError(20001, "bad handler")

// ErrInternalHandler An uncaught and unexpected exception has occurred within the Chatbot implementation.
var ErrInternalHandler = NewActionError(20002, "internal handler")

// Execution Error

// ErrDatabaseError Such as database query failure
var ErrDatabaseError = NewActionError(31001, "database error")

// ErrFilesystemError If reading or writing a file fails, etc.
var ErrFilesystemError = NewActionError(32001, "filesystem error")

// ErrNetworkError e.g. failed to download a file, etc.
var ErrNetworkError = NewActionError(33001, "network error")

// ErrPlatformError e.g. failure to send messages due to bot platform limitations, etc.
var ErrPlatformError = NewActionError(34001, "platform error")

// ErrLoginError Such as trying to send a message to a non-existent user
var ErrLoginError = NewActionError(35001, "login error")

// ErrIAmTired A Chatbot realizes the decision to strike.
var ErrIAmTired = NewActionError(36001, "i am tired")
