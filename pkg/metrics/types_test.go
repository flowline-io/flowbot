package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no change", input: "archive-items", want: "archive-items"},
		{name: "spaces to underscore", input: "my pipeline", want: "my_pipeline"},
		{name: "special chars", input: "a/b?c:d#e", want: "a_b_c_d_e"},
		{name: "truncate long values", input: strings.Repeat("x", 200), want: strings.Repeat("x", 128)},
		{name: "empty string", input: "", want: ""},
		{name: "chinese characters", input: "管道", want: "__"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeLabel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
