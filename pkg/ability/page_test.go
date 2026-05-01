package ability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageRequestDefaults(t *testing.T) {
	pr := PageRequest{}
	assert.Equal(t, 0, pr.Limit)
	assert.Empty(t, pr.Cursor)
	assert.Empty(t, pr.SortBy)
	assert.Empty(t, pr.SortOrder)
}

func TestPageRequestWithValues(t *testing.T) {
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
}

func TestPageInfoDefaults(t *testing.T) {
	pi := PageInfo{}
	assert.Equal(t, 0, pi.Limit)
	assert.False(t, pi.HasMore)
	assert.Empty(t, pi.NextCursor)
	assert.Empty(t, pi.PrevCursor)
	assert.Nil(t, pi.Total)
}

func TestPageInfoWithValues(t *testing.T) {
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
}

func TestPageInfoTotalNil(t *testing.T) {
	pi := PageInfo{}
	assert.Nil(t, pi.Total)
}

func TestListResultEmpty(t *testing.T) {
	lr := ListResult[string]{}
	assert.Nil(t, lr.Items)
	assert.Nil(t, lr.Page)
}

func TestListResultWithItems(t *testing.T) {
	items := []*string{ptr("a"), ptr("b"), ptr("c")}
	lr := ListResult[string]{
		Items: items,
	}
	assert.Len(t, lr.Items, 3)
	assert.Equal(t, "a", *lr.Items[0])
	assert.Equal(t, "b", *lr.Items[1])
	assert.Equal(t, "c", *lr.Items[2])
	assert.Nil(t, lr.Page)
}

func TestListResultWithPage(t *testing.T) {
	total := int64(50)
	lr := ListResult[string]{
		Items: []*string{ptr("x")},
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
}

func TestListResultGenericWithBookmark(t *testing.T) {
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
}

func ptr[T any](v T) *T {
	return &v
}
