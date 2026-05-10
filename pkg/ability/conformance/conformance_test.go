package conformance

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestCanceledContext(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"returns a non-nil canceled context with error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := CanceledContext()
			require.NotNil(t, ctx)
			assert.Error(t, ctx.Err())
		})
	}
}

func TestTestTime(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"returns fixed test timestamp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := TestTime()
			assert.Equal(t, int64(1700000000), tm.Unix())
		})
	}
}

func TestRequireListResult(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts limit and hasMore on list result"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ability.ListResult[ability.Bookmark]{
				Items: []*ability.Bookmark{},
				Page:  &ability.PageInfo{Limit: 10, HasMore: true},
			}
			RequireListResult(t, result, 10, true)
		})
	}
}

func TestRequireTimeoutError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts timeout error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireTimeoutError(t, types.WrapError(types.ErrTimeout, "test", errors.New("canceled")))
		})
	}
}

func TestRequireProviderError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts provider error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireProviderError(t, types.WrapError(types.ErrProvider, "test", errors.New("api down")))
		})
	}
}

func TestRequireInvalidArgError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts invalid argument error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireInvalidArgError(t, types.Errorf(types.ErrInvalidArgument, "field is required"))
		})
	}
}

func TestRequireNotFoundError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts not found error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireNotFoundError(t, types.Errorf(types.ErrNotFound, "item not found"))
		})
	}
}

func TestRequireNotImplementedError(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"asserts not implemented error type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireNotImplementedError(t, types.Errorf(types.ErrNotImplemented, "not implemented"))
		})
	}
}
