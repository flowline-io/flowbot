package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	assert.Equal(t, int64(3), add(1, 2))
	assert.Equal(t, int64(0), add(0, 0))
	assert.Equal(t, int64(-1), add(1, -2))
	assert.Equal(t, int64(0), add(-5, 5))
}
