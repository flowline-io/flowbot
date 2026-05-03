package conformance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/stretchr/testify/require"
)

func TestAssertCursorRoundTrip(t *testing.T) {
	payload := ability.CursorPayload{
		Capability:     "bookmark",
		Backend:        "test",
		ProviderCursor: "next-page",
		Limit:          20,
	}
	AssertCursorRoundTrip(t, CursorSecret, payload)
}

func TestAssertCursorEncoding(t *testing.T) {
	cursor := AssertCursorEncoding(t, CursorSecret, ability.CursorPayload{
		ProviderCursor: "test-cursor",
	})
	require.NotEmpty(t, cursor)
}

func TestAssertPageInfoIsComplete(t *testing.T) {
	AssertPageInfoIsComplete(t, &ability.PageInfo{Limit: 10, HasMore: true, NextCursor: "next"}, 10)
	AssertPageInfoIsComplete(t, &ability.PageInfo{Limit: 0, HasMore: false}, 0)
}

func TestAssertSliceNotNull(t *testing.T) {
	AssertSliceNotNull(t, []*ability.Bookmark{})
	AssertSliceNotNull(t, []*ability.Bookmark{{ID: "1"}})
}
