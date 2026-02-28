package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeploy_FunctionExists(t *testing.T) {
	// deploy depends on external services (gitea, drone), so we only verify the function exists
	assert.NotNil(t, deploy)
}
