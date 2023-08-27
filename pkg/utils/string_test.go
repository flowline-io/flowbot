package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHasHan(t *testing.T) {
	require.True(t, HasHan("hello, 世界"))
	require.False(t, HasHan("hello"))
}

func TestMasker(t *testing.T) {
	require.Equal(t, "qwerty**********7890", Masker("qwertyuiop1234567890", 2))
}

func TestFirstUpper(t *testing.T) {
	require.Equal(t, "Test", FirstUpper("test"))
	require.Equal(t, "", FirstUpper(""))
}

func TestMD5(t *testing.T) {
	type args struct {
		txt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "md5",
			args: args{
				txt: "123456",
			},
			want: "e10adc3949ba59abbe56e057f20f883e",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MD5(tt.args.txt); got != tt.want {
				t.Errorf("MD5() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSHA1(t *testing.T) {
	type args struct {
		txt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "sha1",
			args: args{
				txt: "123456",
			},
			want: "7c4a8d09ca3762af61e59520943dc26494f8941b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SHA1(tt.args.txt); got != tt.want {
				t.Errorf("SHA1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkdownTitle(t *testing.T) {
	type args struct {
		m string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				m: "",
			},
			want: "",
		},
		{
			name: "title",
			args: args{
				m: `# test title
> line 1
> line 2`,
			},
			want: "test title",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MarkdownTitle(tt.args.m); got != tt.want {
				t.Errorf("MarkdownTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
