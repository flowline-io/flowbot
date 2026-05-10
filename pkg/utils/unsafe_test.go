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

func FuzzStringBytesRoundtrip(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("hello world")
	f.Add("\x00\x01\x02\xff")
	f.Add("你好世界")
	f.Add("\u2421")

	f.Fuzz(func(t *testing.T, s string) {
		b := StringToBytes(s)
		got := BytesToString(b)
		if got != s {
			t.Errorf("roundtrip mismatch: input=%q, output=%q", s, got)
		}
	})
}
