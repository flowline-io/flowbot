package utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasHan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		txt  string
		want bool
	}{
		{
			name: "with_chinese",
			txt:  "hello, 世界",
			want: true,
		},
		{
			name: "no_chinese",
			txt:  "hello",
			want: false,
		},
		{
			name: "empty_string",
			txt:  "",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasHan(tt.txt))
		})
	}
}

func TestMasker(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		start int
		want  string
	}{
		{
			name:  "large_string",
			input: "qwertyuiop1234567890",
			start: 2,
			want:  "qwerty**********7890",
		},
		{
			name:  "empty_string",
			input: "",
			start: 0,
			want:  "",
		},
		{
			name:  "start_beyond_length",
			input: "abc",
			start: 5,
			want:  "abc",
		},
		{
			name:  "start_zero",
			input: "abcdefghijk",
			start: 0,
			want:  "abcd***hijk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Masker(tt.input, tt.start))
		})
	}
}

func TestFirstUpper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "lowercase",
			s:    "test",
			want: "Test",
		},
		{
			name: "empty",
			s:    "",
			want: "",
		},
		{
			name: "already_uppercase",
			s:    "Test",
			want: "Test",
		},
		{
			name: "single_char",
			s:    "a",
			want: "A",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FirstUpper(tt.s))
		})
	}
}

func TestMD5(t *testing.T) {
	t.Parallel()
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
		{
			name: "empty_string",
			args: args{
				txt: "",
			},
			want: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name: "unicode_string",
			args: args{
				txt: "hello世界",
			},
			want: "e2c314313e19e12ed756183798a004d6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MD5(tt.args.txt))
		})
	}
}

func TestSHA1(t *testing.T) {
	t.Parallel()
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
		{
			name: "empty_string",
			args: args{
				txt: "",
			},
			want: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name: "unicode_string",
			args: args{
				txt: "hello世界",
			},
			want: "975930521ec6304d233626b03fafddce5c4d6bc7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SHA1(tt.args.txt))
		})
	}
}

func TestMarkdownTitle(t *testing.T) {
	t.Parallel()
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
		{
			name: "title_only_no_body",
			args: args{
				m: "# simple title",
			},
			want: "simple title",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MarkdownTitle(tt.args.m))
		})
	}
}

// TestRandomString tests the RandomString function
func TestRandomString(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got, err := RandomString(tt.n)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if !tt.wantErr {
				assert.Len(t, got, tt.n)
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
						assert.Fail(t, fmt.Sprintf("RandomString() contains invalid character: %c", char))
					}
				}
			}
		})
	}
}

// TestIsUrl tests the IsUrl function
func TestIsUrl(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			assert.Equal(t, tt.want, IsUrl(tt.text))
		})
	}
}

// TestFn tests the Fn function
func TestFn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		public any
		want   string
	}{
		{
			name: "valid_map_with_fn",
			public: map[string]any{
				"fn":   "test_function",
				"name": "test",
			},
			want: "test_function",
		},
		{
			name: "map_without_fn",
			public: map[string]any{
				"name": "test",
			},
			want: "",
		},
		{
			name: "map_with_non_string_fn",
			public: map[string]any{
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
			t.Parallel()
			assert.Equal(t, tt.want, Fn(tt.public))
		})
	}
}

func TestNewUUID(t *testing.T) {
	t.Parallel()

	t.Run("not_empty", func(t *testing.T) {
		t.Parallel()
		assert.NotEmpty(t, NewUUID())
	})

	t.Run("length_36", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, NewUUID(), 36)
	})

	t.Run("format", func(t *testing.T) {
		t.Parallel()
		uuid := NewUUID()
		assert.Equal(t, byte('-'), uuid[8])
		assert.Equal(t, byte('-'), uuid[13])
		assert.Equal(t, byte('-'), uuid[18])
		assert.Equal(t, byte('-'), uuid[23])
	})

	t.Run("unique", func(t *testing.T) {
		t.Parallel()
		uuid1 := NewUUID()
		uuid2 := NewUUID()
		assert.NotEqual(t, uuid1, uuid2)
	})
}

// TestValidImageContentType tests the ValidImageContentType function
func TestValidImageContentType(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			assert.Equal(t, tt.want, ValidImageContentType(tt.ct))
		})
	}
}

// TestFileAndLine tests the FileAndLine function
func TestFileAndLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		check func(t *testing.T, result string)
	}{
		{
			name: "non_empty",
			check: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
			},
		},
		{
			name: "contains_colon",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, ":")
			},
		},
		{
			name: "file_line_format",
			check: func(t *testing.T, result string) {
				parts := strings.Split(result, ":")
				assert.GreaterOrEqual(t, len(parts), 2, "FileAndLine() should have file:line format: %v", result)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FileAndLine()
			tt.check(t, result)
		})
	}
}

// TestYamlToJson tests the YamlToJson function
func TestYamlToJson(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got, err := YamlToJson(tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if !tt.wantErr && got == nil {
				assert.Fail(t, "YamlToJson() returned nil data")
			}
		})
	}
}

func TestBoolToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		b    bool
		want string
	}{
		{
			name: "true",
			b:    true,
			want: "true",
		},
		{
			name: "false",
			b:    false,
			want: "false",
		},
		{
			name: "consistent_output",
			b:    true,
			want: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, BoolToString(tt.b))
		})
	}
}
