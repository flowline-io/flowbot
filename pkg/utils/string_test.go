package utils

import (
	"fmt"
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

func TestSHA256(t *testing.T) {
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
			name: "sha256",
			args: args{
				txt: "123456",
			},
			want: "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92",
		},
		{
			name: "empty_string",
			args: args{
				txt: "",
			},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "unicode_string",
			args: args{
				txt: "hello世界",
			},
			want: "9e83b014808bec990d9d3d35ed8d1a14232929fabf72cb72c1b472f466cc822b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SHA256(tt.args.txt))
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
