package utils

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func slicesEqual(expected, gotten []string) bool {
	if len(expected) != len(gotten) {
		return false
	}
	for i, v := range expected {
		if v != gotten[i] {
			return false
		}
	}
	return true
}

func expectSlicesEqual(t *testing.T, name string, expected, gotten []string) {
	assert.True(t, slicesEqual(expected, gotten), "%s: expected %+v, got %+v", name, expected, gotten)
}

func TestStringSliceDelta(t *testing.T) {
	t.Parallel()
	// Case format:
	// - inputs: old, new
	// - expected outputs: added, removed, intersection
	cases := [][5][]string{
		{
			{"abc", "def", "fff"}, {},
			{}, {"abc", "def", "fff"}, {},
		},
		{
			{}, {}, {}, {}, {},
		},
		{
			{"aa", "xx", "bb", "aa", "bb"}, {"yy", "aa"},
			{"yy"}, {"aa", "bb", "bb", "xx"}, {"aa"},
		},
		{
			{"bb", "aa", "bb"}, {"yy", "aa", "bb", "zzz", "zzz", "cc"},
			{"cc", "yy", "zzz", "zzz"}, {"bb"}, {"aa", "bb"},
		},
		{
			{"aa", "aa", "aa"}, {"aa", "aa", "aa"},
			{}, {}, {"aa", "aa", "aa"},
		},
	}

	for _, tc := range cases {
		added, removed, both := stringSliceDelta(
			tc[0], tc[1],
		)
		expectSlicesEqual(t, "added", tc[2], added)
		expectSlicesEqual(t, "removed", tc[3], removed)
		expectSlicesEqual(t, "both", tc[4], both)
	}
}

// TestToAbsolutePath tests the ToAbsolutePath function
func TestToAbsolutePath(t *testing.T) {
	t.Parallel()
	// Use filepath.Join to create platform-appropriate paths
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{
			name: "relative_path",
			base: filepath.Join("home", "user"),
			path: filepath.Join("documents", "file.txt"),
			want: filepath.Join("home", "user", "documents", "file.txt"),
		},
		{
			name: "current_directory",
			base: filepath.Join("home", "user"),
			path: "./file.txt",
			want: filepath.Join("home", "user", "file.txt"),
		},
		{
			name: "parent_directory",
			base: filepath.Join("home", "user", "projects"),
			path: filepath.Join("..", "documents", "file.txt"),
			want: filepath.Join("home", "user", "documents", "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToAbsolutePath(tt.base, tt.path)
			// Clean both paths to handle platform differences
			got = filepath.Clean(got)
			want := filepath.Clean(tt.want)
			assert.Equal(t, want, got)
		})
	}

	// Test absolute path separately since it behaves differently on Windows vs Unix
	t.Run("absolute_path", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS == "windows" {
			base := `C:\home\user`
			path := `C:\etc\config`
			want := `C:\etc\config`
			got := ToAbsolutePath(base, path)
			assert.Equal(t, want, got)
		} else {
			base := "/home/user"
			path := "/etc/config"
			want := "/etc/config"
			got := ToAbsolutePath(base, path)
			assert.Equal(t, want, got)
		}
	})
}
