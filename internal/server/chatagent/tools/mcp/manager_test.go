package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeEnvOverridesDuplicates(t *testing.T) {
	t.Parallel()
	got := mergeEnv([]string{"PATH=/bin", "HOME=/root"}, map[string]string{"HOME": "/tmp", "FOO": "bar"})
	assert.Equal(t, []string{"PATH=/bin", "HOME=/tmp", "FOO=bar"}, got)
	assert.Equal(t, []string{"PATH=/bin"}, mergeEnv([]string{"PATH=/bin"}, nil))
}
