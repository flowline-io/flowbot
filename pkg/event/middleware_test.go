package event

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func TestRetry_Fields(t *testing.T) {
	t.Parallel()

	retry := Retry{
		MaxRetries:          3,
		InitialInterval:     1 * time.Second,
		MaxInterval:         30 * time.Second,
		Multiplier:          2.0,
		MaxElapsedTime:      2 * time.Minute,
		RandomizationFactor: 0.5,
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "MaxRetries", got: retry.MaxRetries, want: 3},
		{name: "InitialInterval", got: retry.InitialInterval, want: 1 * time.Second},
		{name: "MaxInterval", got: retry.MaxInterval, want: 30 * time.Second},
		{name: "Multiplier", got: retry.Multiplier, want: 2.0},
		{name: "MaxElapsedTime", got: retry.MaxElapsedTime, want: 2 * time.Minute},
		{name: "RandomizationFactor", got: retry.RandomizationFactor, want: 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestRetry_ZeroValues(t *testing.T) {
	t.Parallel()

	retry := Retry{}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "MaxRetries", got: retry.MaxRetries, want: 0},
		{name: "InitialInterval", got: retry.InitialInterval, want: time.Duration(0)},
		{name: "MaxInterval", got: retry.MaxInterval, want: time.Duration(0)},
		{name: "Multiplier", got: retry.Multiplier, want: 0.0},
		{name: "MaxElapsedTime", got: retry.MaxElapsedTime, want: time.Duration(0)},
		{name: "RandomizationFactor", got: retry.RandomizationFactor, want: 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}

	t.Run("hooks and logger are nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, retry.OnRetryHook)
		assert.Nil(t, retry.Logger)
	})
}

func TestRetry_OnRetryHook(t *testing.T) {
	t.Parallel()

	called := false
	retry := Retry{
		MaxRetries: 1,
		OnRetryHook: func(retryNum int, delay time.Duration) {
			called = true
		},
	}

	t.Run("hook is set and callable", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, retry.OnRetryHook)
		retry.OnRetryHook(1, 100*time.Millisecond)
		assert.True(t, called)
	})
}

func TestRetry_Middleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		retry   Retry
		handler func(msg *message.Message) ([]*message.Message, error)
		wantErr bool
		wantLen int
	}{
		{
			name: "success on first attempt",
			retry: Retry{
				MaxRetries:      3,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				Logger:          flog.WatermillLogger,
			},
			handler: func(msg *message.Message) ([]*message.Message, error) {
				return []*message.Message{msg}, nil
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "all retries fail",
			retry: Retry{
				MaxRetries:          1,
				InitialInterval:     10 * time.Millisecond,
				MaxInterval:         50 * time.Millisecond,
				Multiplier:          1.0,
				MaxElapsedTime:      200 * time.Millisecond,
				RandomizationFactor: 0.0,
				Logger:              flog.WatermillLogger,
			},
			handler: func(msg *message.Message) ([]*message.Message, error) {
				return nil, assert.AnError
			},
			wantErr: true,
			wantLen: 0,
		},
		{
			name: "context timeout exhausted",
			retry: Retry{
				MaxRetries:      100,
				MaxElapsedTime:  50 * time.Millisecond,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     20 * time.Millisecond,
				Multiplier:      1.0,
				Logger:          flog.WatermillLogger,
			},
			handler: func(msg *message.Message) ([]*message.Message, error) {
				return nil, assert.AnError
			},
			wantErr: true,
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			middleware := tt.retry.Middleware(tt.handler)
			msg := message.NewMessage("test", []byte("payload"))
			result, err := middleware(msg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestRetry_Middleware_EventualSuccess(t *testing.T) {
	t.Parallel()

	var attempts int
	retry := Retry{
		MaxRetries:          3,
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         50 * time.Millisecond,
		Multiplier:          1.0,
		MaxElapsedTime:      1 * time.Second,
		RandomizationFactor: 0.0,
		Logger:              flog.WatermillLogger,
	}

	handler := func(msg *message.Message) ([]*message.Message, error) {
		attempts++
		if attempts < 3 {
			return nil, assert.AnError
		}
		return []*message.Message{msg}, nil
	}

	middleware := retry.Middleware(handler)
	msg := message.NewMessage("test", []byte("payload"))

	result, err := middleware(msg)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}
