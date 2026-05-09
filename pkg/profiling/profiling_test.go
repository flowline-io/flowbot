package profiling

import (
	"testing"

	"github.com/grafana/pyroscope-go"
)

func TestParseProfileTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []pyroscope.ProfileType
	}{
		{
			name:     "empty returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "single type",
			input:    []string{"cpu"},
			expected: []pyroscope.ProfileType{pyroscope.ProfileCPU},
		},
		{
			name:  "all types",
			input: []string{"cpu", "alloc_objects", "alloc_space", "inuse_objects", "inuse_space", "goroutines", "mutex_count", "mutex_duration", "block_count", "block_duration"},
			expected: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProfileTypes(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestProfileTypeNames(t *testing.T) {
	types := []pyroscope.ProfileType{pyroscope.ProfileCPU, pyroscope.ProfileGoroutines}
	names := profileTypeNames(types)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "cpu" {
		t.Errorf("expected 'cpu', got %q", names[0])
	}
	if names[1] != "goroutines" {
		t.Errorf("expected 'goroutines', got %q", names[1])
	}
}

func TestProfileTypeNamesEmpty(t *testing.T) {
	names := profileTypeNames(nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names for nil, got %d", len(names))
	}
}

func TestPyroscopeLogger(t *testing.T) {
	l := pyroscopeLogger{}
	// Verify the adapter does not panic
	l.Infof("test info %d", 1)
	l.Debugf("test debug %s", "arg")
	l.Errorf("test error %v", "arg")
}
