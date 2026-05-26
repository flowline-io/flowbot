package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoteWebserviceRules_Defined(t *testing.T) {
	tests := []struct {
		name          string
		expectedPaths []string
	}{
		{
			name: "should contain CRUD endpoints",
			expectedPaths: []string{
				"/",    // list and create
				"/:id", // get, update, delete
				"/search",
				"/health",
				"/:id/content", // get and set content
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(map[string]bool)
			for _, r := range noteWebserviceRules {
				paths[r.Path] = true
			}
			for _, expected := range tt.expectedPaths {
				assert.True(t, paths[expected], "expected path %q in note webservice rules", expected)
			}
		})
	}
}

func TestNoteWebserviceRules_NotEmpty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "note webservice rules should not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, noteWebserviceRules)
		})
	}
}
