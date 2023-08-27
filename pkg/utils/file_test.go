package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFileExist(t *testing.T) {
	require.True(t, FileExist("./string.go"))
}
