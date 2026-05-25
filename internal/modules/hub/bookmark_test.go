package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, tagPrompt)
			},
		},
		{
			name: "should contain required sections",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, tagPrompt, "tags")
				assert.Contains(t, tagPrompt, "JSON")
				assert.Contains(t, tagPrompt, "{{.language}}")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestReplaceSimilarTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tags    []string
		similar map[string]string
		want    []string
	}{
		{
			name:    "nil input returns nil",
			tags:    nil,
			similar: map[string]string{"a": "b"},
			want:    nil,
		},
		{
			name:    "empty mapping returns original tags",
			tags:    []string{"go", "rust", "python"},
			similar: map[string]string{},
			want:    []string{"go", "rust", "python"},
		},
		{
			name:    "replaces tag with mapped value",
			tags:    []string{"golang", "rust", "python"},
			similar: map[string]string{"golang": "go"},
			want:    []string{"go", "rust", "python"},
		},
		{
			name:    "deduplicates after mapping",
			tags:    []string{"golang", "go", "rust"},
			similar: map[string]string{"golang": "go"},
			want:    []string{"go", "rust"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := replaceSimilarTags(tt.tags, tt.similar)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSliceEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{name: "equal slices", a: []string{"a", "b"}, b: []string{"a", "b"}, want: true},
		{name: "different elements", a: []string{"a", "b"}, b: []string{"a", "c"}, want: false},
		{name: "different lengths", a: []string{"a"}, b: []string{"a", "b"}, want: false},
		{name: "both empty", a: []string{}, b: []string{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, sliceEqual(tt.a, tt.b))
		})
	}
}

func TestBookmarkCommandRules_Metadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have bookmark list command",
			test: func(t *testing.T) {
				t.Parallel()
				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}
				assert.Contains(t, defines, "bookmark list")
				assert.Equal(t, "newest 10", defines["bookmark list"])
			},
		},
		{
			name: "all command rules should have non-nil handlers",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range commandRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
