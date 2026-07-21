// Package protocol provides platform-agnostic protocol types for request/response handling.
package protocol

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/samber/oops"

	"github.com/flowline-io/flowbot/pkg/types"
)

type ResponseStatus string

const (
	Success ResponseStatus = "ok"
	Failed  ResponseStatus = "failed"
)

const (
	GetLatestEventsAction     = "get_latest_events"
	GetSupportedActionsAction = "get_supported_actions"
	GetStatusAction           = "get_status"
	GetVersionAction          = "get_version"

	SendMessageAction   = "send_message"
	UpdateMessageAction = "update_message"
	DeleteMessageAction = "delete_message"

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
	RetCode string `json:"retcode,omitempty"`
	// Response data
	Data any `json:"data,omitempty"`
	// Error message, it is recommended to fill in a human-readable error message when the action fails to execute,
	// or an empty string when it succeeds.
	Message string `json:"message,omitempty"`
}

func NewError(code int64, message string) oops.OopsErrorBuilder {
	return oops.Code(strconv.FormatInt(code, 10)).
		Public(message)
}

// ErrorCode extracts the string error code from an OopsErrorBuilder created via NewError.
// The returned string is the same value stored in OopsError.Code() when the error is materialized.
func ErrorCode(b oops.OopsErrorBuilder) string {
	err := b.New("")
	var e oops.OopsError
	if errors.As(err, &e) {
		if s, ok := e.Code().(string); ok {
			return s
		}
	}
	return ""
}

// Request Error (10xxx)

// ErrInternalServerError Internal server error
var ErrInternalServerError = NewError(10000, "internal server error")

// ErrBadRequest  Formatting errors (including implementations that do not support MessagePack),
// missing required fields, or incorrect field types
var ErrBadRequest = NewError(10001, "bad request")

// ErrUnsupportedAction The Chatbot implementation does not implement this action
var ErrUnsupportedAction = NewError(10002, "unsupported action")

// ErrBadParam Missing parameter or wrong parameter type
var ErrBadParam = NewError(10003, "bad parameter")

// ErrParamVerificationFailed Missing parameter or wrong parameter type
var ErrParamVerificationFailed = NewError(10031, "parameter verification failed")

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

// ErrNotFound not found
var ErrNotFound = NewError(10009, "not found")

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

// ErrEmitEventError Emit event error
var ErrEmitEventError = NewError(37001, "emit event error")

// ErrShutdownError Shutdown error
var ErrShutdownError = NewError(38001, "shutdown error")

// Retention Error (40xxx to 50xxx)

// Business error (60xxx to 90xxx)

// ErrTokenError missing, invalid or expired access token
var ErrTokenError = NewError(60001, "missing, invalid or expired access token")

// ErrSendMessageFailed Failed to send a message
var ErrSendMessageFailed = NewError(60002, "send message failed")

// ErrFlagError Flag error
var ErrFlagError = NewError(60003, "flag error")

// ErrFlagExpired Flag expired
var ErrFlagExpired = NewError(60004, "flag expired")

// ErrNotAuthorized Not authorized
var ErrNotAuthorized = NewError(60005, "not authorized")

// ErrOAuthError OAuth error
var ErrOAuthError = NewError(60006, "oauth error")

// ErrAccessDenied Access Denied
var ErrAccessDenied = NewError(60007, "access denied")

func NewSuccessResponse(data any) Response {
	return Response{
		Status: Success,
		Data:   data,
	}
}

// NewFailedResponse builds a client-safe failed Response.
// Domain types.Error Message values are returned (intentional API text).
// For ErrInvalidArgument, the wrapped Cause is appended so clients/LLMs can fix bad input.
// oops.Public() is used for protocol builders. Plain/unknown errors stay opaque;
// callers should log the original error before or after converting.
func NewFailedResponse(err error) Response {
	if err == nil {
		return Response{
			Status:  Failed,
			RetCode: "10000",
			Message: "Unknown Error",
		}
	}
	var te *types.Error
	if errors.As(err, &te) {
		return Response{
			Status:  Failed,
			RetCode: domainRetCode(te),
			Message: clientSafeDomainMessage(te),
		}
	}
	var e oops.OopsError
	if errors.As(err, &e) {
		message := e.Public()
		if message == "" {
			message = "Unknown Error"
		}
		return Response{
			Status:  Failed,
			RetCode: fmt.Sprintf("%v", e.Code()),
			Message: message,
		}
	}

	return Response{
		Status:  Failed,
		RetCode: "10000",
		Message: "Unknown Error",
	}
}

// clientSafeDomainMessage returns the intentional API message for a domain error.
// Validation failures (ErrInvalidArgument) include the cause text for repairability.
func clientSafeDomainMessage(te *types.Error) string {
	if te == nil {
		return "Unknown Error"
	}
	message := te.Message
	if message == "" && te.Kind != nil {
		message = te.Kind.Error()
	}
	if message == "" {
		return "Unknown Error"
	}
	if te.Cause == nil || !errors.Is(te.Kind, types.ErrInvalidArgument) {
		return message
	}
	cause := te.Cause.Error()
	if cause == "" || cause == message {
		return message
	}
	// Avoid duplicating when callers already embedded the cause in Message.
	if len(message) >= len(cause) && (message == cause || message[len(message)-len(cause):] == cause) {
		return message
	}
	return message + ": " + cause
}

// domainRetCode maps a types.Error kind to a stable protocol retcode string.
func domainRetCode(te *types.Error) string {
	if te == nil {
		return "10000"
	}
	switch {
	case errors.Is(te.Kind, types.ErrInvalidArgument):
		return "10001"
	case errors.Is(te.Kind, types.ErrUnauthorized):
		return "60005"
	case errors.Is(te.Kind, types.ErrForbidden):
		return "60007"
	case errors.Is(te.Kind, types.ErrNotFound):
		return "10009"
	case errors.Is(te.Kind, types.ErrAlreadyExists), errors.Is(te.Kind, types.ErrConflict):
		return "10010"
	case errors.Is(te.Kind, types.ErrRateLimited):
		return "10011"
	case errors.Is(te.Kind, types.ErrUnavailable):
		return "10012"
	case errors.Is(te.Kind, types.ErrTimeout):
		return "10013"
	case errors.Is(te.Kind, types.ErrNotImplemented):
		return "10002"
	case errors.Is(te.Kind, types.ErrProvider):
		return "10014"
	default:
		return "10000"
	}
}
