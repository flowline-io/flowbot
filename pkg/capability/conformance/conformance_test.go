package conformance

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestCanceledContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"returns a non-nil canceled context with error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CanceledContext()
			require.NotNil(t, ctx)
			assert.Error(t, ctx.Err())
		})
	}
}

func TestTestTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"returns fixed test timestamp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := TestTime()
			assert.Equal(t, int64(1700000000), tm.Unix())
		})
	}
}

func TestRequireListResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts limit and hasMore on list result"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &capability.ListResult[capability.Bookmark]{
				Items: []*capability.Bookmark{},
				Page:  &capability.PageInfo{Limit: 10, HasMore: true},
			}
			RequireListResult(t, result, 10, true)
		})
	}
}

func TestRequireTimeoutError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts timeout error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RequireTimeoutError(t, types.WrapError(types.ErrTimeout, "test", errors.New("canceled")))
		})
	}
}

func TestRequireProviderError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts provider error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RequireProviderError(t, types.WrapError(types.ErrProvider, "test", errors.New("api down")))
		})
	}
}

func TestRequireInvalidArgError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts invalid argument error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RequireInvalidArgError(t, types.Errorf(types.ErrInvalidArgument, "field is required"))
		})
	}
}

func TestRequireNotFoundError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts not found error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RequireNotFoundError(t, types.Errorf(types.ErrNotFound, "item not found"))
		})
	}
}

func TestRequireNotImplementedError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"asserts not implemented error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RequireNotImplementedError(t, types.Errorf(types.ErrNotImplemented, "not implemented"))
		})
	}
}
