package event

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/stretchr/testify/assert"
)

func TestRetry_Fields(t *testing.T) {
	retry := Retry{
		MaxRetries:          3,
		InitialInterval:     1 * time.Second,
		MaxInterval:         30 * time.Second,
		Multiplier:          2.0,
		MaxElapsedTime:      2 * time.Minute,
		RandomizationFactor: 0.5,
	}

	assert.Equal(t, 3, retry.MaxRetries)
	assert.Equal(t, 1*time.Second, retry.InitialInterval)
	assert.Equal(t, 30*time.Second, retry.MaxInterval)
	assert.Equal(t, 2.0, retry.Multiplier)
	assert.Equal(t, 2*time.Minute, retry.MaxElapsedTime)
	assert.Equal(t, 0.5, retry.RandomizationFactor)
}

func TestRetry_ZeroValues(t *testing.T) {
	retry := Retry{}
	assert.Equal(t, 0, retry.MaxRetries)
	assert.Equal(t, time.Duration(0), retry.InitialInterval)
	assert.Equal(t, time.Duration(0), retry.MaxInterval)
	assert.Equal(t, 0.0, retry.Multiplier)
	assert.Equal(t, time.Duration(0), retry.MaxElapsedTime)
	assert.Equal(t, 0.0, retry.RandomizationFactor)
	assert.Nil(t, retry.OnRetryHook)
	assert.Nil(t, retry.Logger)
}

func TestRetry_WithOnRetryHook(t *testing.T) {
	called := false
	retry := Retry{
		MaxRetries:  1,
		OnRetryHook: func(retryNum int, delay time.Duration) { called = true },
	}
	assert.NotNil(t, retry.OnRetryHook)
	retry.OnRetryHook(1, 100*time.Millisecond)
	assert.True(t, called)
}

func TestRetry_Middleware_Success(t *testing.T) {
	retry := Retry{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Logger:          flog.WatermillLogger,
	}

	handler := func(msg *message.Message) ([]*message.Message, error) {
		return []*message.Message{msg}, nil
	}

	middleware := retry.Middleware(handler)
	msg := message.NewMessage("test", []byte("payload"))

	result, err := middleware(msg)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestRetry_Middleware_EventualSuccess(t *testing.T) {
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
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestRetry_Middleware_AllRetriesFail(t *testing.T) {
	retry := Retry{
		MaxRetries:          1,
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         50 * time.Millisecond,
		Multiplier:          1.0,
		MaxElapsedTime:      200 * time.Millisecond,
		RandomizationFactor: 0.0,
		Logger:              flog.WatermillLogger,
	}

	handler := func(msg *message.Message) ([]*message.Message, error) {
		return nil, assert.AnError
	}

	middleware := retry.Middleware(handler)
	msg := message.NewMessage("test", []byte("payload"))

	result, err := middleware(msg)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRetry_Middleware_WithContextTimeout(t *testing.T) {
	retry := Retry{
		MaxRetries:      100,
		MaxElapsedTime:  50 * time.Millisecond,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     20 * time.Millisecond,
		Multiplier:      1.0,
		Logger:          flog.WatermillLogger,
	}

	handler := func(msg *message.Message) ([]*message.Message, error) {
		return nil, assert.AnError
	}

	middleware := retry.Middleware(handler)
	msg := message.NewMessage("test", []byte("payload"))

	_, err := middleware(msg)
	assert.Error(t, err)
}
