package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

// TestRandomString tests the RandomString function
func TestRandomString(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{
			name:    "zero_length",
			n:       0,
			wantErr: false,
		},
		{
			name:    "positive_length",
			n:       10,
			wantErr: false,
		},
		{
			name:    "large_length",
			n:       100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandomString(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("RandomString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.n {
					t.Errorf("RandomString() length = %v, want %v", len(got), tt.n)
				}
				// Check that all characters are from the allowed set
				for _, char := range got {
					found := false
					for _, allowedChar := range letters {
						if char == allowedChar {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("RandomString() contains invalid character: %c", char)
					}
				}
			}
		})
	}
}

// TestIsUrl tests the IsUrl function
func TestIsUrl(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "valid_http",
			text: "http://example.com",
			want: true,
		},
		{
			name: "valid_https",
			text: "https://example.com",
			want: true,
		},
		{
			name: "valid_with_path",
			text: "https://example.com/path/to/resource",
			want: true,
		},
		{
			name: "valid_with_query",
			text: "https://example.com/search?q=test",
			want: true,
		},
		{
			name: "invalid_no_protocol",
			text: "example.com",
			want: false,
		},
		{
			name: "invalid_ftp",
			text: "ftp://example.com",
			want: false,
		},
		{
			name: "invalid_empty",
			text: "",
			want: false,
		},
		{
			name: "invalid_malformed",
			text: "http://",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUrl(tt.text); got != tt.want {
				t.Errorf("IsUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFn tests the Fn function
func TestFn(t *testing.T) {
	tests := []struct {
		name   string
		public interface{}
		want   string
	}{
		{
			name: "valid_map_with_fn",
			public: map[string]interface{}{
				"fn":   "test_function",
				"name": "test",
			},
			want: "test_function",
		},
		{
			name: "map_without_fn",
			public: map[string]interface{}{
				"name": "test",
			},
			want: "",
		},
		{
			name: "map_with_non_string_fn",
			public: map[string]interface{}{
				"fn": 123,
			},
			want: "",
		},
		{
			name:   "non_map_input",
			public: "not a map",
			want:   "",
		},
		{
			name:   "nil_input",
			public: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Fn(tt.public); got != tt.want {
				t.Errorf("Fn() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewUUID tests the NewUUID function
func TestNewUUID(t *testing.T) {
	uuid1 := NewUUID()
	uuid2 := NewUUID()

	// Test that UUIDs are not empty
	if uuid1 == "" {
		t.Error("NewUUID() returned empty string")
	}

	if uuid2 == "" {
		t.Error("NewUUID() returned empty string")
	}

	// Test that UUIDs are different
	if uuid1 == uuid2 {
		t.Error("NewUUID() returned same UUID twice")
	}

	// Test UUID format (basic check for length and hyphens)
	if len(uuid1) != 36 {
		t.Errorf("NewUUID() length = %v, want 36", len(uuid1))
	}

	// Check for hyphens in correct positions
	if uuid1[8] != '-' || uuid1[13] != '-' || uuid1[18] != '-' || uuid1[23] != '-' {
		t.Errorf("NewUUID() format incorrect: %v", uuid1)
	}
}

// TestValidImageContentType tests the ValidImageContentType function
func TestValidImageContentType(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{
			name: "jpeg",
			ct:   "image/jpeg",
			want: true,
		},
		{
			name: "png",
			ct:   "image/png",
			want: true,
		},
		{
			name: "gif",
			ct:   "image/gif",
			want: true,
		},
		{
			name: "webp",
			ct:   "image/webp",
			want: true,
		},
		{
			name: "text",
			ct:   "text/plain",
			want: false,
		},
		{
			name: "video",
			ct:   "video/mp4",
			want: false,
		},
		{
			name: "empty",
			ct:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidImageContentType(tt.ct); got != tt.want {
				t.Errorf("ValidImageContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFileAndLine tests the FileAndLine function
func TestFileAndLine(t *testing.T) {
	result := FileAndLine()

	// Should not be empty
	if result == "" {
		t.Error("FileAndLine() returned empty string")
	}

	// Should contain a colon (separating file and line)
	if !strings.Contains(result, ":") {
		t.Errorf("FileAndLine() should contain colon: %v", result)
	}

	// Should end with a number (line number)
	parts := strings.Split(result, ":")
	if len(parts) < 2 {
		t.Errorf("FileAndLine() should have file:line format: %v", result)
	}
}

// TestYamlToJson tests the YamlToJson function
func TestYamlToJson(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid_yaml",
			data:    []byte("name: test\nage: 30"),
			wantErr: false,
		},
		{
			name:    "empty_yaml",
			data:    []byte(""),
			wantErr: false,
		},
		{
			name:    "invalid_yaml",
			data:    []byte("name: test\n  invalid: yaml: syntax"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YamlToJson(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("YamlToJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("YamlToJson() returned nil data")
			}
		})
	}
}
