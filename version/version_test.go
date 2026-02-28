package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildstamp(t *testing.T) {
	assert.NotEmpty(t, Buildstamp)
}

func TestBuildtags(t *testing.T) {
	assert.NotEmpty(t, Buildtags)
}
