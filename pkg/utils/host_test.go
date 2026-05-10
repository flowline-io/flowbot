package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostInfo(t *testing.T) {
	t.Parallel()
	t.Run("returns valid host info", func(t *testing.T) {
		t.Parallel()
		hostID, hostname, err := HostInfo()

		require.NoError(t, err)
		assert.NotEmpty(t, hostID)
		assert.NotEmpty(t, hostname)
	})
}
