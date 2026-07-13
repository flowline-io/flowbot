package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func TestAssertCursorRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"cursor round trip encodes and decodes correctly"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload := capability.CursorPayload{
				Capability:     "karakeep",
				ProviderCursor: "next-page",
				Limit:          20,
			}
			AssertCursorRoundTrip(t, CursorSecret, payload)
		})
	}
}

func TestAssertCursorEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"assert cursor encoding returns non-empty cursor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cursor := AssertCursorEncoding(t, CursorSecret, capability.CursorPayload{
				ProviderCursor: "test-cursor",
			})
			require.NotEmpty(t, cursor)
		})
	}
}

func TestAssertPageInfoIsComplete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		pi    *capability.PageInfo
		limit int
	}{
		{"page with limit and hasMore", &capability.PageInfo{Limit: 10, HasMore: true, NextCursor: "next"}, 10},
		{"empty page with no limit", &capability.PageInfo{Limit: 0, HasMore: false}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			AssertPageInfoIsComplete(t, tt.pi, tt.limit)
		})
	}
}

func TestAssertSliceNotNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []*capability.Bookmark
	}{
		{"empty slice is not nil", []*capability.Bookmark{}},
		{"populated slice is not nil", []*capability.Bookmark{{ID: "1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			AssertSliceNotNull(t, tt.items)
		})
	}
}
