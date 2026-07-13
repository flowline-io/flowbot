package karakeep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func TestTagsParam_StringSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"string slice tags are returned as-is", []string{"a", "b"}, []string{"a", "b"}},
		{"single element string slice returns correctly", []string{"only"}, []string{"only"}},
		{"multi-element string slice with duplicates returns all", []string{"a", "b", "a"}, []string{"a", "b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := tagsParam(map[string]any{"tags": tt.tags})
			require.NoError(t, err)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestTagsParam_AnySlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"any slice tags are converted to strings", []any{"x", "y"}, []string{"x", "y"}},
		{"single element any slice converted correctly", []any{"single"}, []string{"single"}},
		{"any slice with empty strings returns all", []any{"a", "", "c"}, []string{"a", "", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := tagsParam(map[string]any{"tags": tt.tags})
			require.NoError(t, err)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestTagsParam_AnySliceMixedTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
		want []string
	}{
		{"mixed type any slice is converted to strings", []any{"a", 42, true}, []string{"a", "42", "true"}},
		{"mixed types with float returns correct strings", []any{"a", 3.14, false}, []string{"a", "3.14", "false"}},
		{"mixed types all non-string returns converted strings", []any{1, true, 3.5}, []string{"1", "true", "3.5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := tagsParam(map[string]any{"tags": tt.tags})
			require.NoError(t, err)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestTagsParam_EmptyStringSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
	}{
		{"empty string slice returns error", []string{}},
		{"empty string slice with other params returns error", []string{}},
		{"barely non-empty string slice succeeds", []string{"not-empty"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tagsParam(map[string]any{"tags": tt.tags})
			if tt.name == "barely non-empty string slice succeeds" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "tags are required")
			}
		})
	}
}

func TestTagsParam_EmptyAnySlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
	}{
		{"empty any slice returns error", []any{}},
		{"empty any slice with valid other params returns error", []any{}},
		{"nil slice tags returns error", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := map[string]any{"tags": tt.tags}
			if tt.name == "empty any slice with valid other params returns error" {
				params["other"] = "value"
			}
			_, err := tagsParam(params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "tags are required")
		})
	}
}

func TestTagsParam_Missing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing tags returns error", map[string]any{}},
		{"only other keys present returns error", map[string]any{"other": "val"}},
		{"empty params map with no keys returns error", map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tagsParam(tt.params)
			require.Error(t, err)
		})
	}
}

func TestTagsParam_WrongType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags any
	}{
		{"non-slice type returns error", "not-an-array"},
		{"integer type returns error", 42},
		{"boolean type returns error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tagsParam(map[string]any{"tags": tt.tags})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "must be an array")
		})
	}
}

func TestTagsParam_Nil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"nil tags returns error"},
		{"nil tags in non-empty params returns error"},
		{"nil tags value with other keys present returns error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := map[string]any{"tags": nil}
			if tt.name == "nil tags in non-empty params returns error" {
				params["id"] = "123"
			}
			if tt.name == "nil tags value with other keys present returns error" {
				params["extra"] = true
			}
			_, err := tagsParam(params)
			require.Error(t, err)
		})
	}
}

func TestIDAndTags_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
		wantID string
		want   []string
	}{
		{"valid id and tags returns both values", map[string]any{"id": "bm1", "tags": []string{"tag1"}}, "bm1", []string{"tag1"}},
		{"valid id with any slice tags returns both", map[string]any{"id": "bm2", "tags": []any{"tag2", "tag3"}}, "bm2", []string{"tag2", "tag3"}},
		{"valid id with mixed type tags returns both", map[string]any{"id": "bm3", "tags": []any{"x", 1}}, "bm3", []string{"x", "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, tags, err := idAndTags(tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.want, tags)
		})
	}
}

func TestIDAndTags_MissingID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing id returns error", map[string]any{"tags": []string{"t"}}},
		{"empty id string returns error", map[string]any{"id": "", "tags": []string{"t"}}},
		{"nil id value with valid tags returns error", map[string]any{"id": nil, "tags": []string{"t"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := idAndTags(tt.params)
			require.Error(t, err)
		})
	}
}

func TestIDAndTags_MissingTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing tags returns error", map[string]any{"id": "bm1"}},
		{"nil tags value with valid id returns error", map[string]any{"id": "bm1", "tags": nil}},
		{"wrong type tags with valid id returns error", map[string]any{"id": "bm1", "tags": "wrong"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := idAndTags(tt.params)
			require.Error(t, err)
		})
	}
}

func TestListInvokeResult_Nil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"nil list result returns empty items", "list"},
		{"nil result for get operation returns empty items", "get"},
		{"nil result for search operation returns empty items", "search"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := listInvokeResult(tt.op, nil)
			assert.NotNil(t, result)
			assert.Equal(t, tt.op, result.Operation)
			items, ok := result.Data.([]*capability.Bookmark)
			assert.True(t, ok)
			assert.Empty(t, items)
			assert.NotNil(t, result.Page)
		})
	}
}

func TestListInvokeResult_NonNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		lr   *capability.ListResult[capability.Bookmark]
	}{
		{"non-nil list result preserves items and page", &capability.ListResult[capability.Bookmark]{
			Items: []*capability.Bookmark{{ID: "1", URL: "https://x.com"}},
			Page:  &capability.PageInfo{Limit: 10, HasMore: false},
		}},
		{"non-nil list with multiple items preserves all", &capability.ListResult[capability.Bookmark]{
			Items: []*capability.Bookmark{
				{ID: "1", URL: "https://a.com"},
				{ID: "2", URL: "https://b.com"},
			},
			Page: &capability.PageInfo{Limit: 5, HasMore: true},
		}},
		{"non-nil list with has_more true preserves page fields", &capability.ListResult[capability.Bookmark]{
			Items: []*capability.Bookmark{{ID: "99", URL: "https://z.com"}},
			Page:  &capability.PageInfo{Limit: 50, HasMore: true, NextCursor: "cursor_next"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := listInvokeResult("list", tt.lr)
			items, ok := result.Data.([]*capability.Bookmark)
			require.True(t, ok)
			assert.Len(t, items, len(tt.lr.Items))
			assert.Equal(t, tt.lr.Page.Limit, result.Page.Limit)
			assert.Equal(t, tt.lr.Page.HasMore, result.Page.HasMore)
		})
	}
}
