package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileExist(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "file_exist",
			args: args{
				name: "./string.go",
			},
			want: true,
		},
		{
			name: "file_not_exist",
			args: args{
				name: "./not_exist.test",
			},
			want: false,
		},
		{
			name: "directory_exists",
			args: args{
				name: ".",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FileExist(tt.args.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFileExistEdgeCases tests edge cases for the FileExist function
func TestFileExistEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "empty_filename",
			filename: "",
			want:     false,
		},
		{
			name:     "current_directory",
			filename: ".",
			want:     true,
		},
		{
			name:     "parent_directory",
			filename: "..",
			want:     true,
		},
		{
			name:     "go_mod_file",
			filename: "../../go.mod", // Should exist in project root
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FileExist(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Note: DownloadFile function is difficult to test comprehensively in unit tests
// because it requires network access and external resources.
// Here's a basic test structure that could be expanded with mock servers.

// TestDownloadFileBasic tests basic parameter validation for DownloadFile
func TestDownloadFileBasic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		url      string
		filename string
	}{
		{
			name:     "invalid_url",
			url:      "invalid-url",
			filename: "/tmp/test",
		},
		{
			name:     "empty_filename",
			url:      "http://example.com",
			filename: "",
		},
		{
			name:     "malformed_url_scheme",
			url:      "://badurl",
			filename: "/tmp/test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := DownloadFile(tt.url, tt.filename)
			require.Error(t, err, "DownloadFile() should return error for %s", tt.name)
		})
	}
}
