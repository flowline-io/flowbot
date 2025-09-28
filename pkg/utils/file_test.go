package utils

import (
	"testing"
)

func TestFileExist(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExist(tt.args.name); got != tt.want {
				t.Errorf("FileExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFileExistEdgeCases tests edge cases for the FileExist function
func TestFileExistEdgeCases(t *testing.T) {
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
			got := FileExist(tt.filename)
			if got != tt.want {
				t.Errorf("FileExist(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// Note: DownloadFile function is difficult to test comprehensively in unit tests
// because it requires network access and external resources.
// Here's a basic test structure that could be expanded with mock servers.

// TestDownloadFileBasic tests basic parameter validation for DownloadFile
func TestDownloadFileBasic(t *testing.T) {
	// Test with invalid URL (should return error quickly)
	err := DownloadFile("invalid-url", "/tmp/test")
	if err == nil {
		t.Error("DownloadFile() should return error for invalid URL")
	}

	// Test with empty filename
	err = DownloadFile("http://example.com", "")
	if err == nil {
		t.Error("DownloadFile() should return error for empty filename")
	}
}
