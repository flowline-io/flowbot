package protocol_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
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
		name           string
		err            error
		wantRetCode    string
		wantMsg        string
		mustNotContain string
	}{
		{
			name:        "nil error becomes unknown",
			err:         nil,
			wantRetCode: "10000",
			wantMsg:     "Unknown Error",
		},
		{
			name:           "oops error uses public message only",
			err:            protocol.ErrBadRequest.New("missing field"),
			wantRetCode:    "10001",
			wantMsg:        "bad request",
			mustNotContain: "missing field",
		},
		{
			name:           "plain error does not leak detail",
			err:            errors.New("secret connection string"),
			wantRetCode:    "10000",
			wantMsg:        "Unknown Error",
			mustNotContain: "secret connection string",
		},
		{
			name:        "domain not found keeps intentional message",
			err:         types.Errorf(types.ErrNotFound, "bookmark %s missing", "abc"),
			wantRetCode: "10009",
			wantMsg:     "bookmark abc missing",
		},
		{
			name:           "domain wrap does not leak cause text",
			err:            types.WrapError(types.ErrProvider, "upstream call failed", errors.New("dsn=secret")),
			wantRetCode:    "10014",
			wantMsg:        "upstream call failed",
			mustNotContain: "dsn=secret",
		},
		{
			name:        "domain kind-only uses kind text",
			err:         &types.Error{Kind: types.ErrForbidden},
			wantRetCode: "60007",
			wantMsg:     "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := protocol.NewFailedResponse(tt.err)
			assert.Equal(t, protocol.Failed, resp.Status)
			assert.Equal(t, tt.wantRetCode, resp.RetCode)
			assert.Equal(t, tt.wantMsg, resp.Message)
			if tt.mustNotContain != "" {
				assert.NotContains(t, resp.Message, tt.mustNotContain)
			}
			require.NotEqual(t, protocol.Success, resp.Status)
		})
	}
}
