package bookmark

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
