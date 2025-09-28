package utils

import (
	"path/filepath"
	"runtime"
	"testing"
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
	if !slicesEqual(expected, gotten) {
		t.Errorf("%s: expected %+v, got %+v", name, expected, gotten)
	}
}

func TestStringSliceDelta(t *testing.T) {
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

// TestIsNullValue tests the IsNullValue function
func TestIsNullValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{
			name:  "null_value_string",
			input: "\u2421",
			want:  true,
		},
		{
			name:  "empty_string",
			input: "",
			want:  false,
		},
		{
			name:  "regular_string",
			input: "hello",
			want:  false,
		},
		{
			name:  "non_string_input",
			input: 123,
			want:  false,
		},
		{
			name:  "nil_input",
			input: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNullValue(tt.input); got != tt.want {
				t.Errorf("IsNullValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseVersionPart tests the ParseVersionPart function
func TestParseVersionPart(t *testing.T) {
	tests := []struct {
		name string
		vers string
		want int
	}{
		{
			name: "simple_number",
			vers: "1",
			want: 1,
		},
		{
			name: "number_with_suffix",
			vers: "123abc",
			want: 123,
		},
		{
			name: "zero",
			vers: "0",
			want: 0,
		},
		{
			name: "empty_string",
			vers: "",
			want: 0,
		},
		{
			name: "non_numeric",
			vers: "abc",
			want: 0,
		},
		{
			name: "large_number",
			vers: "9999",
			want: 0, // Should be 0 because it exceeds 0x1fff (8191)
		},
		{
			name: "max_valid",
			vers: "8191",
			want: 8191,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseVersionPart(tt.vers); got != tt.want {
				t.Errorf("ParseVersionPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseVersion tests the ParseVersion function
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name string
		vers string
		want int
	}{
		{
			name: "simple_version",
			vers: "1.2.3",
			want: (1 << 16) | (2 << 8) | 3,
		},
		{
			name: "version_with_prefix",
			vers: "v1.2.3",
			want: (1 << 16) | (2 << 8) | 3,
		},
		{
			name: "major_minor_only",
			vers: "1.2",
			want: (1 << 16) | (2 << 8),
		},
		{
			name: "major_only",
			vers: "1",
			want: 1 << 16,
		},
		{
			name: "empty_string",
			vers: "",
			want: 0,
		},
		{
			name: "version_with_suffix",
			vers: "1.2.3-alpha",
			want: (1 << 16) | (2 << 8) | 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseVersion(tt.vers); got != tt.want {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBase10Version tests the Base10Version function
func TestBase10Version(t *testing.T) {
	tests := []struct {
		name string
		hex  int
		want int64
	}{
		{
			name: "version_1_2_3",
			hex:  (1 << 16) | (2 << 8) | 3,
			want: 10203,
		},
		{
			name: "version_0_0_0",
			hex:  0,
			want: 0,
		},
		{
			name: "version_10_20_30",
			hex:  (10 << 16) | (20 << 8) | 30,
			want: 102030,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Base10Version(tt.hex); got != tt.want {
				t.Errorf("Base10Version() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVersionCompare tests the VersionCompare function
func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   int
		v2   int
		want int
	}{
		{
			name: "v1_greater_than_v2",
			v1:   ParseVersion("2.0.0"),
			v2:   ParseVersion("1.0.0"),
			want: 256, // Difference in major version (2-1) << 8
		},
		{
			name: "v1_equal_to_v2",
			v1:   ParseVersion("1.2.3"),
			v2:   ParseVersion("1.2.5"), // Patch version ignored
			want: 0,
		},
		{
			name: "v1_less_than_v2",
			v1:   ParseVersion("1.0.0"),
			v2:   ParseVersion("2.0.0"),
			want: -256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionCompare(tt.v1, tt.v2); got != tt.want {
				t.Errorf("VersionCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMax tests the Max function
func TestMax(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "a_greater_than_b",
			a:    10,
			b:    5,
			want: 10,
		},
		{
			name: "b_greater_than_a",
			a:    3,
			b:    7,
			want: 7,
		},
		{
			name: "a_equals_b",
			a:    5,
			b:    5,
			want: 5,
		},
		{
			name: "negative_numbers",
			a:    -3,
			b:    -1,
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Max(tt.a, tt.b); got != tt.want {
				t.Errorf("Max() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestToAbsolutePath tests the ToAbsolutePath function
func TestToAbsolutePath(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToAbsolutePath(tt.base, tt.path)
			// Clean both paths to handle platform differences
			got = filepath.Clean(got)
			want := filepath.Clean(tt.want)
			if got != want {
				t.Errorf("ToAbsolutePath() = %v, want %v", got, want)
			}
		})
	}

	// Test absolute path separately since it behaves differently on Windows vs Unix
	t.Run("absolute_path", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			base := `C:\home\user`
			path := `C:\etc\config`
			want := `C:\etc\config`
			got := ToAbsolutePath(base, path)
			if got != want {
				t.Errorf("ToAbsolutePath() = %v, want %v", got, want)
			}
		} else {
			base := "/home/user"
			path := "/etc/config"
			want := "/etc/config"
			got := ToAbsolutePath(base, path)
			if got != want {
				t.Errorf("ToAbsolutePath() = %v, want %v", got, want)
			}
		}
	})
}

// TestMergeMaps tests the MergeMaps function
func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name        string
		dst         map[string]any
		src         map[string]any
		wantChanged bool
	}{
		{
			name:        "empty_source",
			dst:         map[string]any{"key": "value"},
			src:         map[string]any{},
			wantChanged: false,
		},
		{
			name:        "nil_destination",
			dst:         nil,
			src:         map[string]any{"key": "value"},
			wantChanged: true,
		},
		{
			name:        "simple_merge",
			dst:         map[string]any{"key1": "value1"},
			src:         map[string]any{"key2": "value2"},
			wantChanged: true,
		},
		{
			name: "nested_map_merge",
			dst: map[string]any{
				"nested": map[string]any{
					"key1": "value1",
				},
			},
			src: map[string]any{
				"nested": map[string]any{
					"key2": "value2",
				},
			},
			wantChanged: true,
		},
		{
			name:        "null_value_deletion",
			dst:         map[string]any{"key1": "value1", "key2": "value2"},
			src:         map[string]any{"key1": "\u2421"}, // null value
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := MergeMaps(tt.dst, tt.src)

			if changed != tt.wantChanged {
				t.Errorf("MergeMaps() changed = %v, want %v", changed, tt.wantChanged)
			}

			if result == nil && tt.src != nil && len(tt.src) > 0 {
				t.Error("MergeMaps() should not return nil result when there's source data")
			}
		})
	}
}
