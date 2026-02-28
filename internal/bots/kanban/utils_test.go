package kanban

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := map[string]any{
		"name": "test",
		"age":  25,
	}

	var result testStruct
	err := unmarshal(input, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 25, result.Age)
}

func TestUnmarshal_EmptyMap(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}

	var result testStruct
	err := unmarshal(map[string]any{}, &result)
	assert.NoError(t, err)
	assert.Empty(t, result.Name)
}
