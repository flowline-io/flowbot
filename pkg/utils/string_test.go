package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasHan(t *testing.T) {
	t.Parallel()
	require.True(t, HasHan("hello, 世界"))
	require.False(t, HasHan("hello"))
}

func TestMasker(t *testing.T) {
	t.Parallel()
	require.Equal(t, "qwerty**********7890", Masker("qwertyuiop1234567890", 2))
}

func TestFirstUpper(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Test", FirstUpper("test"))
	require.Empty(t, FirstUpper(""))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MD5(tt.args.txt); got != tt.want {
				t.Errorf("MD5() = %v, want %v", got, tt.want)
			}
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SHA1(tt.args.txt); got != tt.want {
				t.Errorf("SHA1() = %v, want %v", got, tt.want)
			}
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MarkdownTitle(tt.args.m); got != tt.want {
				t.Errorf("MarkdownTitle() = %v, want %v", got, tt.want)
			}
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
			if got := IsUrl(tt.text); got != tt.want {
				t.Errorf("IsUrl() = %v, want %v", got, tt.want)
			}
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

func FuzzHasHan(f *testing.F) {
	f.Add("hello")
	f.Add("你好")
	f.Add("hello, 世界")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		got := HasHan(s)
		if got && len(s) == 0 {
			t.Error("HasHan() = true for empty string")
		}
		// ASCII-only strings should not have Han characters
		ascii := true
		for _, r := range s {
			if r > 127 {
				ascii = false
				break
			}
		}
		if ascii && got {
			t.Errorf("HasHan(%q) = true for pure ASCII string", s)
		}
	})
}

func FuzzIsUrl(f *testing.F) {
	f.Add("https://example.com")
	f.Add("http://example.com/path?q=1")
	f.Add("")
	f.Add("not a url")
	f.Add("ftp://example.com")

	f.Fuzz(func(t *testing.T, s string) {
		if r := recover(); r != nil {
			t.Fatalf("IsUrl panicked on %q: %v", s, r)
		}
		_ = IsUrl(s)
	})
}

func FuzzFirstUpper(f *testing.F) {
	f.Add("test")
	f.Add("")
	f.Add("Test")
	f.Add("hello world")

	f.Fuzz(func(t *testing.T, s string) {
		got := FirstUpper(s)
		if s == "" && got != "" {
			t.Errorf("FirstUpper(%q) = %q, expected empty", s, got)
		}
		if s != "" && got == "" {
			t.Errorf("FirstUpper(%q) = %q, expected non-empty", s, got)
		}
	})
}

func FuzzMasker(f *testing.F) {
	f.Add("qwertyuiop1234567890", 2)
	f.Add("abc", 0)
	f.Add("short", 10)
	f.Add("", 0)

	f.Fuzz(func(t *testing.T, input string, start int) {
		got := Masker(input, start)
		if len(got) != len(input) && input != "" && start < len(input) {
			t.Errorf("Masker(%q, %d) len=%d, expected len=%d", input, start, len(got), len(input))
		}
	})
}

func FuzzMD5(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("123456")

	f.Fuzz(func(t *testing.T, s string) {
		got := MD5(s)
		if len(got) != 32 {
			t.Errorf("MD5(%q) = %q, len=%d, expected 32", s, got, len(got))
		}
		if got2 := MD5(s); got != got2 {
			t.Errorf("MD5(%q) not deterministic: %q vs %q", s, got, got2)
		}
	})
}

func FuzzSHA1(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("123456")

	f.Fuzz(func(t *testing.T, s string) {
		got := SHA1(s)
		if len(got) != 40 {
			t.Errorf("SHA1(%q) = %q, len=%d, expected 40", s, got, len(got))
		}
		if got2 := SHA1(s); got != got2 {
			t.Errorf("SHA1(%q) not deterministic: %q vs %q", s, got, got2)
		}
	})
}

func FuzzMarkdownTitle(f *testing.F) {
	f.Add("")
	f.Add("# title")
	f.Add("## subtitle")
	f.Add("plain text")
	f.Add("# title\ncontent")

	f.Fuzz(func(t *testing.T, s string) {
		if r := recover(); r != nil {
			t.Fatalf("MarkdownTitle panicked on %q: %v", s, r)
		}
		_ = MarkdownTitle(s)
	})
}

func FuzzValidImageContentType(f *testing.F) {
	f.Add("image/png")
	f.Add("text/plain")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		if r := recover(); r != nil {
			t.Fatalf("ValidImageContentType panicked on %q: %v", s, r)
		}
		_ = ValidImageContentType(s)
	})
}

func FuzzYamlToJson(f *testing.F) {
	f.Add([]byte("name: test"))
	f.Add([]byte(""))
	f.Add([]byte("name: test\nage: 30"))
	f.Add([]byte("key: value\nlist:\n  - a\n  - b"))

	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := YamlToJson(data)
		if err == nil && got == nil && len(data) > 0 {
			t.Errorf("YamlToJson(%q) returned nil result with no error", data)
		}
	})
}

func FuzzFn(f *testing.F) {
	f.Add("test")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		if r := recover(); r != nil {
			t.Fatalf("Fn panicked on %q: %v", s, r)
		}
		_ = Fn(s)

		_ = Fn(map[string]any{"fn": s})
		_ = Fn(map[string]any{"fn": 123})
	})
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BoolToString(tt.b); got != tt.want {
				t.Errorf("BoolToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
