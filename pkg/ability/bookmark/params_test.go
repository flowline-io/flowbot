package bookmark

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStringer struct{ val string }

func (t testStringer) String() string { return t.val }

func TestStringParam_String(t *testing.T) {
	v, ok := stringParam(map[string]any{"key": "hello"}, "key")
	assert.True(t, ok)
	assert.Equal(t, "hello", v)
}

func TestStringParam_Stringer(t *testing.T) {
	v, ok := stringParam(map[string]any{"key": testStringer{"world"}}, "key")
	assert.True(t, ok)
	assert.Equal(t, "world", v)
}

func TestStringParam_Fallback(t *testing.T) {
	v, ok := stringParam(map[string]any{"key": 42}, "key")
	assert.True(t, ok)
	assert.Equal(t, "42", v)
}

func TestStringParam_Missing(t *testing.T) {
	_, ok := stringParam(map[string]any{}, "key")
	assert.False(t, ok)
}

func TestStringParam_NilValue(t *testing.T) {
	_, ok := stringParam(map[string]any{"key": nil}, "key")
	assert.False(t, ok)
}

func TestIntParam_Int(t *testing.T) {
	v, ok := intParam(map[string]any{"key": 42}, "key")
	assert.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestIntParam_Int64(t *testing.T) {
	v, ok := intParam(map[string]any{"key": int64(100)}, "key")
	assert.True(t, ok)
	assert.Equal(t, 100, v)
}

func TestIntParam_Float64(t *testing.T) {
	v, ok := intParam(map[string]any{"key": float64(99.9)}, "key")
	assert.True(t, ok)
	assert.Equal(t, 99, v)
}

func TestIntParam_StringValid(t *testing.T) {
	v, ok := intParam(map[string]any{"key": "42"}, "key")
	assert.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestIntParam_StringInvalid(t *testing.T) {
	_, ok := intParam(map[string]any{"key": "abc"}, "key")
	assert.False(t, ok)
}

func TestIntParam_Other(t *testing.T) {
	_, ok := intParam(map[string]any{"key": []int{1}}, "key")
	assert.False(t, ok)
}

func TestIntParam_Missing(t *testing.T) {
	_, ok := intParam(map[string]any{}, "key")
	assert.False(t, ok)
}

func TestBoolParam_Bool(t *testing.T) {
	v, ok := boolParam(map[string]any{"key": true}, "key")
	assert.True(t, ok)
	assert.True(t, v)

	v, ok = boolParam(map[string]any{"key": false}, "key")
	assert.True(t, ok)
	assert.False(t, v)
}

func TestBoolParam_String(t *testing.T) {
	tests := []struct {
		input string
		want  bool
		ok    bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"1", true, true},
		{"0", false, true},
		{"t", true, true},
		{"f", false, true},
		{"yes", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			v, ok := boolParam(map[string]any{"key": tc.input}, "key")
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.want, v)
			}
		})
	}
}

func TestBoolParam_Other(t *testing.T) {
	_, ok := boolParam(map[string]any{"key": 42}, "key")
	assert.False(t, ok)
}

func TestBoolParam_Missing(t *testing.T) {
	_, ok := boolParam(map[string]any{}, "key")
	assert.False(t, ok)
}

func TestRequiredString_Present(t *testing.T) {
	v, err := requiredString(map[string]any{"id": "abc"}, "id")
	require.NoError(t, err)
	assert.Equal(t, "abc", v)
}

func TestRequiredString_Missing(t *testing.T) {
	_, err := requiredString(map[string]any{}, "id")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "id is required"))
}

func TestRequiredString_Empty(t *testing.T) {
	_, err := requiredString(map[string]any{"id": ""}, "id")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "id is required"))
}

func TestTagsParam_StringSlice(t *testing.T) {
	v, err := tagsParam(map[string]any{"tags": []string{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, v)
}

func TestTagsParam_AnySlice(t *testing.T) {
	v, err := tagsParam(map[string]any{"tags": []any{"x", "y"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, v)
}

func TestTagsParam_AnySliceMixedTypes(t *testing.T) {
	v, err := tagsParam(map[string]any{"tags": []any{"a", 42, true}})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "42", "true"}, v)
}

func TestTagsParam_EmptyStringSlice(t *testing.T) {
	_, err := tagsParam(map[string]any{"tags": []string{}})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "tags are required"))
}

func TestTagsParam_EmptyAnySlice(t *testing.T) {
	_, err := tagsParam(map[string]any{"tags": []any{}})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "tags are required"))
}

func TestTagsParam_Missing(t *testing.T) {
	_, err := tagsParam(map[string]any{})
	require.Error(t, err)
}

func TestTagsParam_WrongType(t *testing.T) {
	_, err := tagsParam(map[string]any{"tags": "not-an-array"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "must be an array"))
}

func TestTagsParam_Nil(t *testing.T) {
	_, err := tagsParam(map[string]any{"tags": nil})
	require.Error(t, err)
}

func TestIDAndTags_Valid(t *testing.T) {
	id, tags, err := idAndTags(map[string]any{"id": "bm1", "tags": []string{"tag1"}})
	require.NoError(t, err)
	assert.Equal(t, "bm1", id)
	assert.Equal(t, []string{"tag1"}, tags)
}

func TestIDAndTags_MissingID(t *testing.T) {
	_, _, err := idAndTags(map[string]any{"tags": []string{"t"}})
	require.Error(t, err)
}

func TestIDAndTags_MissingTags(t *testing.T) {
	_, _, err := idAndTags(map[string]any{"id": "bm1"})
	require.Error(t, err)
}

func TestPageRequestFromParams(t *testing.T) {
	pr := pageRequestFromParams(map[string]any{
		"limit":      10,
		"cursor":     "abc123",
		"sort_by":    "created_at",
		"sort_order": "desc",
	})
	assert.Equal(t, 10, pr.Limit)
	assert.Equal(t, "abc123", pr.Cursor)
	assert.Equal(t, "created_at", pr.SortBy)
	assert.Equal(t, "desc", pr.SortOrder)

	pr = pageRequestFromParams(map[string]any{})
	assert.Equal(t, 0, pr.Limit)
	assert.Equal(t, "", pr.Cursor)
}

func TestListInvokeResult_Nil(t *testing.T) {
	result := listInvokeResult("list", nil)
	assert.NotNil(t, result)
	assert.Equal(t, "list", result.Operation)
	items, ok := result.Data.([]*ability.Bookmark)
	assert.True(t, ok)
	assert.Empty(t, items)
	assert.NotNil(t, result.Page)
}

func TestListInvokeResult_NonNil(t *testing.T) {
	bms := &ability.ListResult[ability.Bookmark]{
		Items: []*ability.Bookmark{{ID: "1", URL: "https://x.com"}},
		Page:  &ability.PageInfo{Limit: 10, HasMore: false},
	}
	result := listInvokeResult("list", bms)
	items := result.Data.([]*ability.Bookmark)
	assert.Len(t, items, 1)
	assert.Equal(t, 10, result.Page.Limit)
}
