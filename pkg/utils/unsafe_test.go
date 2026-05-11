package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		want []byte
	}{
		{
			name: "simple_string",
			s:    "test",
			want: []byte("test"),
		},
		{
			name: "empty_string",
			s:    "",
			want: []byte(nil),
		},
		{
			name: "string_with_special_chars",
			s:    "hello\x00world",
			want: []byte("hello\x00world"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, StringToBytes(tt.s))
		})
	}
}

func TestBytesToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		b    []byte
		want string
	}{
		{
			name: "simple_bytes",
			b:    []byte("test"),
			want: "test",
		},
		{
			name: "empty_bytes",
			b:    []byte(""),
			want: "",
		},
		{
			name: "bytes_with_special_chars",
			b:    []byte("hello\x00world"),
			want: "hello\x00world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, BytesToString(tt.b))
		})
	}
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
		assert.Equal(t, s, got, "roundtrip mismatch: input=%q, output=%q", s, got)
	})
}
