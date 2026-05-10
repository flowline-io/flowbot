package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostInfo(t *testing.T) {
	t.Run("returns valid host info", func(t *testing.T) {
		hostID, hostname, err := HostInfo()

		require.NoError(t, err)
		assert.NotEmpty(t, hostID)
		assert.NotEmpty(t, hostname)
	})
}
