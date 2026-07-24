package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/utils"
)

func TestWordCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty", content: "", want: 0},
		{name: "one word", content: "hello", want: 1},
		{name: "two words", content: "hello world", want: 2},
		{name: "newlines", content: "a\nb\nc", want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, utils.WordCount(tt.content))
		})
	}
}
