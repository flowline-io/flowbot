package ability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageRequestDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"zero value page request has default fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pr := PageRequest{}
			assert.Equal(t, 0, pr.Limit)
			assert.Empty(t, pr.Cursor)
			assert.Empty(t, pr.SortBy)
			assert.Empty(t, pr.SortOrder)
		})
	}
}

func TestPageRequestWithValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"page request with values preserves fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pr := PageRequest{
				Limit:     10,
				Cursor:    "abc123",
				SortBy:    "created_at",
				SortOrder: "desc",
			}
			assert.Equal(t, 10, pr.Limit)
			assert.Equal(t, "abc123", pr.Cursor)
			assert.Equal(t, "created_at", pr.SortBy)
			assert.Equal(t, "desc", pr.SortOrder)
		})
	}
}

func TestPageInfoDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"zero value page info has default fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pi := PageInfo{}
			assert.Equal(t, 0, pi.Limit)
			assert.False(t, pi.HasMore)
			assert.Empty(t, pi.NextCursor)
			assert.Empty(t, pi.PrevCursor)
			assert.Nil(t, pi.Total)
		})
	}
}

func TestPageInfoWithValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"page info with values preserves fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			total := int64(100)
			pi := PageInfo{
				Limit:      20,
				HasMore:    true,
				NextCursor: "cursor_next",
				PrevCursor: "cursor_prev",
				Total:      &total,
			}
			assert.Equal(t, 20, pi.Limit)
			assert.True(t, pi.HasMore)
			assert.Equal(t, "cursor_next", pi.NextCursor)
			assert.Equal(t, "cursor_prev", pi.PrevCursor)
			assert.NotNil(t, pi.Total)
			assert.Equal(t, int64(100), *pi.Total)
		})
	}
}

func TestPageInfoTotalNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"page info total is nil by default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pi := PageInfo{}
			assert.Nil(t, pi.Total)
		})
	}
}

func TestListResultEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"empty list result has nil items and page"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lr := ListResult[string]{}
			assert.Nil(t, lr.Items)
			assert.Nil(t, lr.Page)
		})
	}
}

func TestListResultWithItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"list result with items preserves values"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := []*string{new("a"), new("b"), new("c")}
			lr := ListResult[string]{
				Items: items,
			}
			assert.Len(t, lr.Items, 3)
			assert.Equal(t, "a", *lr.Items[0])
			assert.Equal(t, "b", *lr.Items[1])
			assert.Equal(t, "c", *lr.Items[2])
			assert.Nil(t, lr.Page)
		})
	}
}

func TestListResultWithPage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"list result with page preserves page fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			total := int64(50)
			lr := ListResult[string]{
				Items: []*string{new("x")},
				Page: &PageInfo{
					Limit:      10,
					HasMore:    true,
					NextCursor: "next",
					Total:      &total,
				},
			}
			assert.Len(t, lr.Items, 1)
			assert.Equal(t, "x", *lr.Items[0])
			assert.NotNil(t, lr.Page)
			assert.Equal(t, 10, lr.Page.Limit)
			assert.True(t, lr.Page.HasMore)
			assert.Equal(t, "next", lr.Page.NextCursor)
		})
	}
}

func TestListResultGenericWithBookmark(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"list result generic with bookmark preserves fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bm := Bookmark{
				ID:         "1",
				URL:        "https://example.com",
				Title:      "Example",
				Archived:   false,
				Favourited: true,
			}
			lr := ListResult[Bookmark]{
				Items: []*Bookmark{&bm},
			}
			assert.Len(t, lr.Items, 1)
			assert.Equal(t, "https://example.com", lr.Items[0].URL)
			assert.True(t, lr.Items[0].Favourited)
		})
	}
}

//go:fix inline
func ptr[T any](v T) *T {
	return new(v)
}
