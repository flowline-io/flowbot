package utils

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", want: 0},
		{name: "invalid", err: types.ErrInvalidArgument, want: 2},
		{name: "unauthenticated", err: types.ErrUnauthorized, want: 3},
		{name: "denied", err: types.ErrForbidden, want: 3},
		{name: "not found", err: types.ErrNotFound, want: 4},
		{name: "conflict", err: types.ErrConflict, want: 5},
		{name: "exists", err: types.ErrAlreadyExists, want: 5},
		{name: "rate", err: types.ErrRateLimited, want: 6},
		{name: "unavailable", err: types.ErrUnavailable, want: 6},
		{name: "timeout", err: types.ErrTimeout, want: 6},
		{name: "provider retryable", err: &types.Error{Kind: types.ErrProvider, Message: "upstream", Retryable: true}, want: 6},
		{name: "provider not retryable", err: types.ErrProvider, want: 1},
		{name: "not implemented", err: types.ErrNotImplemented, want: 8},
		{name: "internal", err: types.ErrInternal, want: 1},
		{name: "run failed", err: types.ErrInvokeRun, want: 1},
		{name: "cancelled", err: ErrCancelled, want: 1},
		{name: "confirmation", err: ErrConfirmationRequired, want: 7},
		{name: "api not found", err: &client.APIError{StatusCode: 404, RetCode: "10009", Message: "missing"}, want: 4},
		{name: "usage", err: errors.New("unknown flag: --nope"), want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExitCode(tt.err))
		})
	}
}

func TestErrorObject(t *testing.T) {
	t.Parallel()
	cancelled := ErrorObject(ErrCancelled)
	assert.Equal(t, "cancelled", cancelled["code"])
	assert.Equal(t, false, cancelled["retryable"])
	assert.NotContains(t, cancelled["hint"], "--yes")

	need := ErrorObject(ErrConfirmationRequired)
	assert.Equal(t, "confirmation_required", need["code"])
	assert.Contains(t, need["hint"], "--yes")

	conflict := ErrorObject(types.ErrConflict)
	assert.Equal(t, "conflict", conflict["code"])
	assert.Equal(t, true, conflict["retryable"])
}

func TestWriteEnvelope(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, WriteEnvelope(&buf, map[string]string{"id": "a"}, nil, false))
	assert.Contains(t, buf.String(), "{\n  \"data\": {")
	assert.NotContains(t, buf.String(), "\"page\"")

	buf.Reset()
	listed := map[string]any{
		"data": []any{},
		"page": map[string]any{"limit": 20, "has_more": false},
	}
	require.NoError(t, WriteEnvelope(&buf, listed, nil, false))
	var got map[string]any
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &got))
	_, dataIsObject := got["data"].(map[string]any)
	assert.False(t, dataIsObject)
	assert.Contains(t, got, "page")

	buf.Reset()
	var items []string
	require.NoError(t, WriteEnvelope(&buf, items, nil, false))
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, []any{}, got["data"])
}

func TestPrintUsesCommandStdout(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().String("output", "json", "")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	require.NoError(t, Print(cmd, map[string]string{"id": "1"}, nil))
	assert.Contains(t, buf.String(), "\"data\"")
}

func TestNormalizeUnavailableKeepsRetryAfter(t *testing.T) {
	t.Parallel()
	err := Normalize(&client.APIError{
		StatusCode: 503,
		RetCode:    "10012",
		Message:    "idempotency key is already in progress",
		RetryAfter: time.Second,
	})
	var te *types.Error
	require.ErrorAs(t, err, &te)
	require.ErrorIs(t, te.Kind, types.ErrUnavailable)
	require.Equal(t, int64(1000), te.RetryAfter.Milliseconds())
}
