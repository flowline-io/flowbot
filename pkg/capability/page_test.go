package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageRequestDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		assert func(*testing.T, PageRequest)
	}{
		{"zero value page request has default fields", func(t *testing.T, pr PageRequest) {
			assert.Equal(t, 0, pr.Limit)
			assert.Empty(t, pr.Cursor)
			assert.Empty(t, pr.SortBy)
			assert.Empty(t, pr.SortOrder)
		}},
		{"zero value has 0 limit", func(t *testing.T, pr PageRequest) {
			assert.Zero(t, pr.Limit)
		}},
		{"zero value has empty sort by", func(t *testing.T, pr PageRequest) {
			assert.Empty(t, pr.SortBy)
			assert.Empty(t, pr.SortOrder)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, PageRequest{})
		})
	}
}

func TestPageRequestWithValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		limit     int
		cursor    string
		sortBy    string
		sortOrder string
	}{
		{"page request with values preserves fields", 10, "abc123", "created_at", "desc"},
		{"page request with max limit preserves fields", 1000, "cursor_max", "updated_at", "asc"},
		{"page request with empty cursor preserves fields", 5, "", "title", "asc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pr := PageRequest{
				Limit:     tt.limit,
				Cursor:    tt.cursor,
				SortBy:    tt.sortBy,
				SortOrder: tt.sortOrder,
			}
			assert.Equal(t, tt.limit, pr.Limit)
			assert.Equal(t, tt.cursor, pr.Cursor)
			assert.Equal(t, tt.sortBy, pr.SortBy)
			assert.Equal(t, tt.sortOrder, pr.SortOrder)
		})
	}
}

func TestPageInfoDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		assert func(*testing.T, PageInfo)
	}{
		{"zero value page info has default fields", func(t *testing.T, pi PageInfo) {
			assert.Equal(t, 0, pi.Limit)
			assert.False(t, pi.HasMore)
			assert.Empty(t, pi.NextCursor)
			assert.Empty(t, pi.PrevCursor)
			assert.Nil(t, pi.Total)
		}},
		{"zero value has false has_more", func(t *testing.T, pi PageInfo) {
			assert.False(t, pi.HasMore)
		}},
		{"zero value has nil total pointer", func(t *testing.T, pi PageInfo) {
			assert.Nil(t, pi.Total)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, PageInfo{})
		})
	}
}

func TestPageInfoWithValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		limit      int
		hasMore    bool
		nextCursor string
		prevCursor string
		total      int64
		totalNil   bool
	}{
		{"page info with values preserves fields", 20, true, "cursor_next", "cursor_prev", 100, false},
		{"page info with false has_more preserves fields", 10, false, "next", "prev", 50, false},
		{"page info with zero total pointer preserves zero", 5, false, "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var totalPtr *int64
			if !tt.totalNil {
				t := tt.total
				totalPtr = &t
			}
			pi := PageInfo{
				Limit:      tt.limit,
				HasMore:    tt.hasMore,
				NextCursor: tt.nextCursor,
				PrevCursor: tt.prevCursor,
				Total:      totalPtr,
			}
			assert.Equal(t, tt.limit, pi.Limit)
			assert.Equal(t, tt.hasMore, pi.HasMore)
			assert.Equal(t, tt.nextCursor, pi.NextCursor)
			assert.Equal(t, tt.prevCursor, pi.PrevCursor)
			if tt.totalNil {
				assert.Nil(t, pi.Total)
			} else {
				require.NotNil(t, pi.Total)
				assert.Equal(t, tt.total, *pi.Total)
			}
		})
	}
}

func TestPageInfoTotalNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setupFn func() *PageInfo
	}{
		{"page info total is nil by default", func() *PageInfo { return &PageInfo{} }},
		{"page info with explicit nil total stays nil", func() *PageInfo { return &PageInfo{Total: nil} }},
		{"page info with non-nil total pointer has correct value", func() *PageInfo {
			t := int64(42)
			return &PageInfo{Total: &t}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pi := tt.setupFn()
			if tt.name == "page info with non-nil total pointer has correct value" {
				assert.NotNil(t, pi.Total)
				assert.Equal(t, int64(42), *pi.Total)
			} else {
				assert.Nil(t, pi.Total)
			}
		})
	}
}

func TestListResultEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		itemsNil bool
		pageNil  bool
		lr       ListResult[string]
	}{
		{"empty list result has nil items and page", true, true, ListResult[string]{}},
		{"list result with items but nil page preserves nil page", false, true, ListResult[string]{Items: []*string{new("x")}}},
		{"list result with page but nil items preserves nil items", true, false, ListResult[string]{Page: &PageInfo{Limit: 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.itemsNil {
				assert.Nil(t, tt.lr.Items)
			} else {
				assert.NotNil(t, tt.lr.Items)
			}
			if tt.pageNil {
				assert.Nil(t, tt.lr.Page)
			} else {
				assert.NotNil(t, tt.lr.Page)
			}
		})
	}
}

func TestListResultWithItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []*string
	}{
		{"list result with items preserves values", []*string{new("a"), new("b"), new("c")}},
		{"single item list result preserves value", []*string{new("only")}},
		{"empty slice items returns non-nil but empty items", []*string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lr := ListResult[string]{
				Items: tt.items,
			}
			assert.NotNil(t, lr.Items)
			assert.Len(t, lr.Items, len(tt.items))
			for i, item := range tt.items {
				assert.Equal(t, *item, *lr.Items[i])
			}
			assert.Nil(t, lr.Page)
		})
	}
}

func TestListResultWithPage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []*string
		page  *PageInfo
	}{
		{"list result with page preserves page fields", []*string{new("x")}, &PageInfo{
			Limit: 10, HasMore: true, NextCursor: "next", Total: new(int64(50)),
		}},
		{"list result with page but no items preserves page", nil, &PageInfo{
			Limit: 5, HasMore: true, NextCursor: "nxt",
		}},
		{"list result with page has_more=false preserves fields", []*string{new("a"), new("b")}, &PageInfo{
			Limit: 20, HasMore: false, PrevCursor: "prev",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lr := ListResult[string]{
				Items: tt.items,
				Page:  tt.page,
			}
			assert.Equal(t, tt.items, lr.Items)
			assert.NotNil(t, lr.Page)
			assert.Equal(t, tt.page.Limit, lr.Page.Limit)
			assert.Equal(t, tt.page.HasMore, lr.Page.HasMore)
			assert.Equal(t, tt.page.NextCursor, lr.Page.NextCursor)
			assert.Equal(t, tt.page.PrevCursor, lr.Page.PrevCursor)
		})
	}
}

func TestListResultGenericWithBookmark(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []*Bookmark
	}{
		{"list result generic with bookmark preserves fields", []*Bookmark{{
			ID: "1", URL: "https://example.com", Title: "Example", Archived: false, Favourited: true,
		}}},
		{"list result with multiple bookmarks preserves all fields", []*Bookmark{
			{ID: "1", URL: "https://a.com", Title: "A", Archived: false, Favourited: true},
			{ID: "2", URL: "https://b.com", Title: "B", Archived: true, Favourited: false},
			{ID: "3", URL: "https://c.com", Title: "C", Archived: false, Favourited: false},
		}},
		{"list result with archived bookmark preserves archived flag", []*Bookmark{{
			ID: "99", URL: "https://old.example.com", Title: "Old", Archived: true, Favourited: false,
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lr := ListResult[Bookmark]{
				Items: tt.items,
			}
			assert.Len(t, lr.Items, len(tt.items))
			for i, item := range tt.items {
				assert.Equal(t, item.URL, lr.Items[i].URL)
				assert.Equal(t, item.Favourited, lr.Items[i].Favourited)
				assert.Equal(t, item.Archived, lr.Items[i].Archived)
			}
		})
	}
}
