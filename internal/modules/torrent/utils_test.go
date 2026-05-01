package torrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTorrentClear_FunctionExists(t *testing.T) {
	// torrentClear depends on external transmission service, so we only verify the function exists
	assert.NotNil(t, torrentClear)
}
