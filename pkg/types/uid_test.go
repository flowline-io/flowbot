package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUid_IsZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		uid  Uid
		want bool
	}{
		{
			name: "ZeroUid is zero",
			uid:  ZeroUid,
			want: true,
		},
		{
			name: "empty string is zero",
			uid:  Uid(""),
			want: true,
		},
		{
			name: "non-empty uid is not zero",
			uid:  Uid("user-1"),
			want: false,
		},
		{
			name: "single character is not zero",
			uid:  Uid("x"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.uid.IsZero())
		})
	}
}

func TestUid_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		uid  Uid
		want string
	}{
		{
			name: "normal uid",
			uid:  Uid("test-id"),
			want: "test-id",
		},
		{
			name: "empty uid",
			uid:  Uid(""),
			want: "",
		},
		{
			name: "ZeroUid",
			uid:  ZeroUid,
			want: string(ZeroUid),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.uid.String())
		})
	}
}
