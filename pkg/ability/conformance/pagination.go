package conformance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertCursorRoundTrip verifies that a cursor can be encoded and decoded correctly.
func AssertCursorRoundTrip(t *testing.T, secret []byte, payload ability.CursorPayload) {
	t.Helper()
	cursor, err := ability.EncodeCursor(secret, payload)
	require.NoError(t, err, "EncodeCursor must not return an error")
	require.NotEmpty(t, cursor, "encoded cursor must not be empty")

	decoded, err := ability.DecodeCursor(secret, cursor, TestTime())
	require.NoError(t, err, "DecodeCursor must succeed for a valid cursor")
	assert.Equal(t, payload.ProviderCursor, decoded.ProviderCursor, "provider cursor must round-trip")
}

// AssertCursorEncoding verifies that encoding a cursor with valid data produces a non-empty string.
func AssertCursorEncoding(t *testing.T, secret []byte, payload ability.CursorPayload) string {
	t.Helper()
	cursor, err := ability.EncodeCursor(secret, payload)
	require.NoError(t, err, "EncodeCursor must not return an error")
	require.NotEmpty(t, cursor, "encoded cursor must not be empty")
	return cursor
}

// AssertPageInfoIsComplete verifies all fields in PageInfo are set correctly.
func AssertPageInfoIsComplete(t *testing.T, page *ability.PageInfo, limit int) {
	t.Helper()
	require.NotNil(t, page, "PageInfo must not be nil")
	assert.GreaterOrEqual(t, page.Limit, 0, "Limit must be >= 0")
	if page.HasMore {
		assert.NotEmpty(t, page.NextCursor, "NextCursor must be set when HasMore is true")
	} else {
		assert.Empty(t, page.NextCursor, "NextCursor must be empty when HasMore is false")
	}
}

// AssertSliceNotNull verifies that a slice is not nil.
func AssertSliceNotNull[T any](t *testing.T, items []*T) {
	t.Helper()
	assert.NotNil(t, items, "slice must not be nil, use empty slice instead")
}
