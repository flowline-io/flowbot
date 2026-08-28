package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			assert.Error(t, tt.err)
		})
	}
}

func TestError_Error_Nil(t *testing.T) {
	t.Parallel()
	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		var e *Error
		assert.Equal(t, "error", e.Error())
	})
}

func TestError_Error_Message(t *testing.T) {
	t.Parallel()
	t.Run("with message", func(t *testing.T) {
		t.Parallel()
		e := &Error{Message: "something went wrong"}
		assert.Equal(t, "something went wrong", e.Error())
	})
}

func TestError_Error_Cause(t *testing.T) {
	t.Parallel()
	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		e := &Error{Cause: errors.New("root cause")}
		assert.Equal(t, "root cause", e.Error())
	})
}

func TestError_Error_Kind(t *testing.T) {
	t.Parallel()
	t.Run("with kind", func(t *testing.T) {
		t.Parallel()
		e := &Error{Kind: ErrNotFound}
		assert.Equal(t, "not found", e.Error())
	})
}

func TestError_Error_Empty(t *testing.T) {
	t.Parallel()
	t.Run("empty error", func(t *testing.T) {
		t.Parallel()
		e := &Error{}
		assert.Equal(t, "error", e.Error())
	})
}

func TestError_Unwrap_Nil(t *testing.T) {
	t.Parallel()
	t.Run("nil unwrap", func(t *testing.T) {
		t.Parallel()
		var e *Error
		assert.NoError(t, e.Unwrap())
	})
}

func TestError_Unwrap_WithCause(t *testing.T) {
	t.Parallel()
	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("root")
		e := &Error{Cause: cause}
		assert.Equal(t, cause, e.Unwrap())
	})
}

func TestError_Unwrap_WithoutCause(t *testing.T) {
	t.Parallel()
	t.Run("without cause", func(t *testing.T) {
		t.Parallel()
		e := &Error{}
		assert.NoError(t, e.Unwrap())
	})
}

func TestError_Is_Match(t *testing.T) {
	t.Parallel()
	t.Run("matches kind", func(t *testing.T) {
		t.Parallel()
		e := &Error{Kind: ErrNotFound}
		assert.True(t, e.Is(ErrNotFound))
	})
}

func TestError_Is_NoMatch(t *testing.T) {
	t.Parallel()
	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		e := &Error{Kind: ErrNotFound}
		assert.False(t, e.Is(ErrAlreadyExists))
	})
}

func TestError_Is_Nil(t *testing.T) {
	t.Parallel()
	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		var e *Error
		assert.False(t, e.Is(ErrNotFound))
	})
}

func TestWrapError(t *testing.T) {
	t.Parallel()
	t.Run("wrap error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("root cause")
		err := WrapError(ErrProvider, "provider failed", cause)

		require.ErrorIs(t, err, ErrProvider)
		assert.Equal(t, "provider failed: root cause", err.Error())

		var fe *Error
		require.ErrorAs(t, err, &fe)
		assert.Equal(t, ErrProvider, fe.Kind)
		assert.Equal(t, cause, fe.Cause)
	})
}

func TestErrorf(t *testing.T) {
	t.Parallel()
	t.Run("errorf", func(t *testing.T) {
		t.Parallel()
		err := Errorf(ErrInvalidArgument, "field %s is required", "id")

		require.ErrorIs(t, err, ErrInvalidArgument)
		assert.Equal(t, "field id is required", err.Error())

		var fe *Error
		require.ErrorAs(t, err, &fe)
		assert.Equal(t, ErrInvalidArgument, fe.Kind)
	})
}

func TestError_Error_JoinsCause(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "message and cause",
			err:  &Error{Message: "kanboard create task", Cause: errors.New("failed to create task, json: cannot unmarshal bool into Go value of type int64")},
			want: "kanboard create task: failed to create task, json: cannot unmarshal bool into Go value of type int64",
		},
		{
			name: "message already ends with cause",
			err:  &Error{Message: "invalid yaml: already detailed", Cause: errors.New("already detailed")},
			want: "invalid yaml: already detailed",
		},
		{
			name: "kind and cause without message",
			err:  &Error{Kind: ErrProvider, Cause: errors.New("api down")},
			want: "provider error: api down",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestClientMessage_OmitsCause(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "domain wrap omits cause",
			err:  WrapError(ErrProvider, "kanboard create task", errors.New("json-rpc false")),
			want: "kanboard create task",
		},
		{
			name: "plain error falls back to Error",
			err:  errors.New("plain"),
			want: "plain",
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ClientMessage(tt.err))
		})
	}
}
