package protocol_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestNewErrorAndErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     int64
		message  string
		wantCode string
	}{
		{
			name:     "internal server error code",
			code:     10000,
			message:  "internal server error",
			wantCode: "10000",
		},
		{
			name:     "bad request code",
			code:     10001,
			message:  "bad request",
			wantCode: "10001",
		},
		{
			name:     "token error code",
			code:     60001,
			message:  "token error",
			wantCode: "60001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builder := protocol.NewError(tt.code, tt.message)
			assert.Equal(t, tt.wantCode, protocol.ErrorCode(builder))
		})
	}
}

func TestNewSuccessResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data any
	}{
		{name: "string data", data: "ok"},
		{name: "map data", data: map[string]any{"id": 1}},
		{name: "nil data", data: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := protocol.NewSuccessResponse(tt.data)
			assert.Equal(t, protocol.Success, resp.Status)
			assert.Equal(t, tt.data, resp.Data)
			assert.Empty(t, resp.RetCode)
		})
	}
}

func TestNewFailedResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantRetCode string
		wantMsgSub  string
	}{
		{
			name:        "nil error becomes unknown",
			err:         nil,
			wantRetCode: "10000",
			wantMsgSub:  "Unknown Error",
		},
		{
			name:        "oops error uses code and public message",
			err:         protocol.ErrBadRequest.New("missing field"),
			wantRetCode: "10001",
			wantMsgSub:  "bad request",
		},
		{
			name:        "plain error falls back to 10000",
			err:         errors.New("plain failure"),
			wantRetCode: "10000",
			wantMsgSub:  "plain failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := protocol.NewFailedResponse(tt.err)
			assert.Equal(t, protocol.Failed, resp.Status)
			assert.Equal(t, tt.wantRetCode, resp.RetCode)
			assert.Contains(t, resp.Message, tt.wantMsgSub)
			require.NotEqual(t, protocol.Success, resp.Status)
		})
	}
}
