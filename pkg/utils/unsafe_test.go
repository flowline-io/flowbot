package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringToBytes(t *testing.T) {
	require.Equal(t, []byte("test"), StringToBytes("test"))
}

func TestBytesToString(t *testing.T) {
	require.Equal(t, "test", BytesToString([]byte("test")))
}
