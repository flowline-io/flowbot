package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateSessionPreview(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{name: "empty", text: " \n\t ", limit: 40, want: ""},
		{name: "collapses whitespace", text: "hello   world", limit: 40, want: "hello world"},
		{name: "truncates runes", text: "一二三四五六七八九十", limit: 5, want: "一二三四…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncateSessionPreview(tt.text, tt.limit))
		})
	}
}
