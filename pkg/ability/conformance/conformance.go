// Package conformance provides a standard test suite for ability adapters.
// Any new provider backed by an ability Service interface must pass these
// tests to guarantee consistent pagination and error handling.
package conformance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

// CanceledContext returns a context that is already canceled.
func CanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// CursorSecret is a shared secret for conformance cursor tests.
var CursorSecret = []byte("conformance-cursor-test-secret-v1")

// TestTime is a deterministic timestamp for cursor-based conformance tests.
func TestTime() time.Time {
	return time.Unix(1700000000, 0)
}

// RequireListResult checks that a ListResult has the expected structure
// including non-nil Items, non-nil Page, correct Limit, and HasMore logic.
func RequireListResult[T any](t *testing.T, result *ability.ListResult[T], limit int, hasMore bool) {
	t.Helper()
	require.NotNil(t, result, "ListResult must not be nil")
	require.NotNil(t, result.Items, "Items must not be nil (use empty slice)")
	require.NotNil(t, result.Page, "Page must not be nil")
	require.Equal(t, limit, result.Page.Limit, "Page.Limit must match the requested limit")
	require.Equal(t, hasMore, result.Page.HasMore, "Page.HasMore must reflect whether there are more items")
}

// RequireTimeoutError checks that err wraps types.ErrTimeout.
func RequireTimeoutError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected error from canceled context")
	require.True(t, errors.Is(err, types.ErrTimeout), "error must wrap ErrTimeout, got: %v", err)
}

// RequireProviderError checks that err wraps types.ErrProvider.
func RequireProviderError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected error from provider failure")
	require.True(t, errors.Is(err, types.ErrProvider), "error must wrap ErrProvider, got: %v", err)
}

// RequireInvalidArgError checks that err wraps types.ErrInvalidArgument.
func RequireInvalidArgError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected error for invalid/missing input")
	require.True(t, errors.Is(err, types.ErrInvalidArgument), "error must wrap ErrInvalidArgument, got: %v", err)
}

// RequireNotFoundError checks that err wraps types.ErrNotFound.
func RequireNotFoundError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected error for missing entity")
	require.True(t, errors.Is(err, types.ErrNotFound), "error must wrap ErrNotFound, got: %v", err)
}

// RequireNotImplementedError checks that err wraps types.ErrNotImplemented.
func RequireNotImplementedError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected error for not-implemented operation")
	require.True(t, errors.Is(err, types.ErrNotImplemented), "error must wrap ErrNotImplemented, got: %v", err)
}
