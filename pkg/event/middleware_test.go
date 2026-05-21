package event

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func TestBackoffMiddleware_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "success_with_string_payload", payload: []byte("payload")},
		{name: "success_with_empty_payload", payload: []byte("")},
		{name: "success_with_json_payload", payload: []byte(`{"key":"value"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := backoff.Config{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     50 * time.Millisecond,
			}
			mw := backoff.Middleware(cfg, flog.WatermillLogger)
			handler := func(msg *message.Message) ([]*message.Message, error) {
				return []*message.Message{msg}, nil
			}
			msg := message.NewMessage("test", tt.payload)
			result, err := mw(handler)(msg)
			require.NoError(t, err)
			assert.Len(t, result, 1)
		})
	}
}

func TestBackoffMiddleware_Failure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		maxRetries int
	}{
		{name: "exhausts_2_attempts", maxRetries: 2},
		{name: "exhausts_3_attempts", maxRetries: 3},
		{name: "fails_on_single_attempt", maxRetries: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := backoff.Config{
				MaxAttempts:     tt.maxRetries,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     50 * time.Millisecond,
			}
			mw := backoff.Middleware(cfg, flog.WatermillLogger)
			handler := func(_ *message.Message) ([]*message.Message, error) {
				return nil, assert.AnError
			}
			msg := message.NewMessage("test", []byte("payload"))
			_, err := mw(handler)(msg)
			require.Error(t, err)
		})
	}
}

func TestBackoffMiddleware_EventualSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		successAfter int
	}{
		{name: "succeeds_after_2_failures", successAfter: 3},
		{name: "succeeds_after_1_failure", successAfter: 2},
		{name: "succeeds_on_first_attempt", successAfter: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var attempts int
			cfg := backoff.Config{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     50 * time.Millisecond,
			}
			mw := backoff.Middleware(cfg, flog.WatermillLogger)
			handler := func(msg *message.Message) ([]*message.Message, error) {
				attempts++
				if attempts < tt.successAfter {
					return nil, assert.AnError
				}
				return []*message.Message{msg}, nil
			}
			msg := message.NewMessage("test", []byte("payload"))
			result, err := mw(handler)(msg)
			require.NoError(t, err)
			assert.Len(t, result, 1)
		})
	}
}
