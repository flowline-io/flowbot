package server

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestDomainErrorStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "ErrInvalidArgument returns StatusBadRequest",
			err:        types.ErrInvalidArgument,
			wantStatus: fiber.StatusBadRequest,
			wantOK:     true,
		},
		{
			name:       "ErrUnauthorized returns StatusUnauthorized",
			err:        types.ErrUnauthorized,
			wantStatus: fiber.StatusUnauthorized,
			wantOK:     true,
		},
		{
			name:       "ErrForbidden returns StatusForbidden",
			err:        types.ErrForbidden,
			wantStatus: fiber.StatusForbidden,
			wantOK:     true,
		},
		{
			name:       "ErrNotFound returns StatusNotFound",
			err:        types.ErrNotFound,
			wantStatus: fiber.StatusNotFound,
			wantOK:     true,
		},
		{
			name:       "ErrAlreadyExists returns StatusConflict",
			err:        types.ErrAlreadyExists,
			wantStatus: fiber.StatusConflict,
			wantOK:     true,
		},
		{
			name:       "ErrConflict returns StatusConflict",
			err:        types.ErrConflict,
			wantStatus: fiber.StatusConflict,
			wantOK:     true,
		},
		{
			name:       "ErrRateLimited returns StatusTooManyRequests",
			err:        types.ErrRateLimited,
			wantStatus: fiber.StatusTooManyRequests,
			wantOK:     true,
		},
		{
			name:       "ErrUnavailable returns StatusServiceUnavailable",
			err:        types.ErrUnavailable,
			wantStatus: fiber.StatusServiceUnavailable,
			wantOK:     true,
		},
		{
			name:       "ErrTimeout returns StatusGatewayTimeout",
			err:        types.ErrTimeout,
			wantStatus: fiber.StatusGatewayTimeout,
			wantOK:     true,
		},
		{
			name:       "ErrNotImplemented returns StatusNotImplemented",
			err:        types.ErrNotImplemented,
			wantStatus: fiber.StatusNotImplemented,
			wantOK:     true,
		},
		{
			name:       "ErrProvider returns StatusBadGateway",
			err:        types.ErrProvider,
			wantStatus: fiber.StatusBadGateway,
			wantOK:     true,
		},
		{
			name:       "ErrInternal returns StatusInternalServerError",
			err:        types.ErrInternal,
			wantStatus: fiber.StatusInternalServerError,
			wantOK:     true,
		},
		{
			name:       "unknown error returns StatusInternalServerError with false",
			err:        errors.New("some random error"),
			wantStatus: fiber.StatusInternalServerError,
			wantOK:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, ok := domainErrorStatus(tt.err)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestDomainErrorStatus_WrappedError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "wrapped ErrNotFound is detected",
			err:        types.Errorf(types.ErrNotFound, "item %s not found", "xyz"),
			wantStatus: fiber.StatusNotFound,
			wantOK:     true,
		},
		{
			name:       "wrapped ErrForbidden is detected",
			err:        types.Errorf(types.ErrForbidden, "access denied to %s", "resource"),
			wantStatus: fiber.StatusForbidden,
			wantOK:     true,
		},
		{
			name:       "wrapped ErrUnauthorized is detected",
			err:        types.Errorf(types.ErrUnauthorized, "token expired"),
			wantStatus: fiber.StatusUnauthorized,
			wantOK:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, ok := domainErrorStatus(tt.err)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
