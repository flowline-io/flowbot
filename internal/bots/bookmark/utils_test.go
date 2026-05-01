package bookmark

import (
	"testing"

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
