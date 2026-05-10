package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidArgument", ErrInvalidArgument},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrForbidden", ErrForbidden},
		{"ErrNotFound", ErrNotFound},
		{"ErrAlreadyExists", ErrAlreadyExists},
		{"ErrConflict", ErrConflict},
		{"ErrRateLimited", ErrRateLimited},
		{"ErrUnavailable", ErrUnavailable},
		{"ErrTimeout", ErrTimeout},
		{"ErrNotImplemented", ErrNotImplemented},
		{"ErrProvider", ErrProvider},
		{"ErrInternal", ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
		})
	}
}

func TestError_Error_Nil(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		var e *Error
		assert.Equal(t, "error", e.Error())
	})
}

func TestError_Error_Message(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		e := &Error{Message: "something went wrong"}
		assert.Equal(t, "something went wrong", e.Error())
	})
}

func TestError_Error_Cause(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		e := &Error{Cause: errors.New("root cause")}
		assert.Equal(t, "root cause", e.Error())
	})
}

func TestError_Error_Kind(t *testing.T) {
	t.Run("with kind", func(t *testing.T) {
		e := &Error{Kind: ErrNotFound}
		assert.Equal(t, "not found", e.Error())
	})
}

func TestError_Error_Empty(t *testing.T) {
	t.Run("empty error", func(t *testing.T) {
		e := &Error{}
		assert.Equal(t, "error", e.Error())
	})
}

func TestError_Unwrap_Nil(t *testing.T) {
	t.Run("nil unwrap", func(t *testing.T) {
		var e *Error
		assert.Nil(t, e.Unwrap())
	})
}

func TestError_Unwrap_WithCause(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("root")
		e := &Error{Cause: cause}
		assert.Equal(t, cause, e.Unwrap())
	})
}

func TestError_Unwrap_WithoutCause(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		e := &Error{}
		assert.Nil(t, e.Unwrap())
	})
}

func TestError_Is_Match(t *testing.T) {
	t.Run("matches kind", func(t *testing.T) {
		e := &Error{Kind: ErrNotFound}
		assert.True(t, e.Is(ErrNotFound))
	})
}

func TestError_Is_NoMatch(t *testing.T) {
	t.Run("no match", func(t *testing.T) {
		e := &Error{Kind: ErrNotFound}
		assert.False(t, e.Is(ErrAlreadyExists))
	})
}

func TestError_Is_Nil(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		var e *Error
		assert.False(t, e.Is(ErrNotFound))
	})
}

func TestWrapError(t *testing.T) {
	t.Run("wrap error", func(t *testing.T) {
		cause := errors.New("root cause")
		err := WrapError(ErrProvider, "provider failed", cause)

		assert.True(t, errors.Is(err, ErrProvider))
		assert.Equal(t, "provider failed", err.Error())

		var fe *Error
		require.True(t, errors.As(err, &fe))
		assert.Equal(t, ErrProvider, fe.Kind)
		assert.Equal(t, cause, fe.Cause)
	})
}

func TestErrorf(t *testing.T) {
	t.Run("errorf", func(t *testing.T) {
		err := Errorf(ErrInvalidArgument, "field %s is required", "id")

		assert.True(t, errors.Is(err, ErrInvalidArgument))
		assert.Equal(t, "field id is required", err.Error())

		var fe *Error
		require.True(t, errors.As(err, &fe))
		assert.Equal(t, ErrInvalidArgument, fe.Kind)
	})
}

func TestError_Error_MessageOverridesCause(t *testing.T) {
	t.Run("message overrides cause", func(t *testing.T) {
		e := &Error{Message: "wrapped error", Cause: errors.New("root")}
		assert.Equal(t, "wrapped error", e.Error())
	})
}
