package kanban

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name     string
		input    map[string]any
		expected testStruct
	}{
		{
			name: "full struct",
			input: map[string]any{
				"name": "test",
				"age":  25,
			},
			expected: testStruct{Name: "test", Age: 25},
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: testStruct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var result testStruct
			err := unmarshal(tt.input, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Name, result.Name)
			if tt.expected.Age != 0 {
				assert.Equal(t, tt.expected.Age, result.Age)
			}
			if tt.name == "empty map" {
				assert.Empty(t, result.Name)
			}
		})
	}
}
