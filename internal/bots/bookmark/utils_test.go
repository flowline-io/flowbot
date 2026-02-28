package bookmark

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/stretchr/testify/assert"
)

func TestReplaceSimilarTags_EmptyInput(t *testing.T) {
	result := replaceSimilarTags(nil, map[string]string{"a": "b"})
	assert.Nil(t, result)
}

func TestReplaceSimilarTags_NoMapping(t *testing.T) {
	tags := []string{"go", "rust", "python"}
	result := replaceSimilarTags(tags, map[string]string{})
	assert.Equal(t, tags, result)
}

func TestReplaceSimilarTags_WithMapping(t *testing.T) {
	tags := []string{"golang", "rust", "python"}
	similar := map[string]string{"golang": "go"}
	result := replaceSimilarTags(tags, similar)
	assert.Equal(t, []string{"go", "rust", "python"}, result)
}

func TestReplaceSimilarTags_DeduplicatesAfterMapping(t *testing.T) {
	tags := []string{"golang", "go", "rust"}
	similar := map[string]string{"golang": "go"}
	result := replaceSimilarTags(tags, similar)
	assert.Equal(t, []string{"go", "rust"}, result)
}

func TestSliceEqual_Equal(t *testing.T) {
	assert.True(t, sliceEqual([]string{"a", "b"}, []string{"a", "b"}))
}

func TestSliceEqual_NotEqual(t *testing.T) {
	assert.False(t, sliceEqual([]string{"a", "b"}, []string{"a", "c"}))
}

func TestSliceEqual_DifferentLengths(t *testing.T) {
	assert.False(t, sliceEqual([]string{"a"}, []string{"a", "b"}))
}

func TestSliceEqual_Empty(t *testing.T) {
	assert.True(t, sliceEqual([]string{}, []string{}))
}

func TestConvertTagsToStrings(t *testing.T) {
	tags := []hoarder.Tag{
		{Name: "foo"},
		{Name: "bar"},
	}
	result := convertTagsToStrings(tags)
	assert.Equal(t, []string{"foo", "bar"}, result)
}

func TestConvertTagsToStrings_Empty(t *testing.T) {
	result := convertTagsToStrings([]hoarder.Tag{})
	assert.Empty(t, result)
}

func TestConvertBookmarkTagsToStrings(t *testing.T) {
	tags := []hoarder.BookmarkTagsInner{
		{Name: "alpha"},
		{Name: "beta"},
	}
	result := convertBookmarkTagsToStrings(tags)
	assert.Equal(t, []string{"alpha", "beta"}, result)
}

func TestConvertStringsToBookmarkTags(t *testing.T) {
	tags := []string{"alpha", "beta"}
	result := convertStringsToBookmarkTags(tags)
	assert.Len(t, result, 2)
	assert.Equal(t, "alpha", result[0].Name)
	assert.Equal(t, "beta", result[1].Name)
}
