package profiling

import (
	"testing"

	"github.com/grafana/pyroscope-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProfileTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []pyroscope.ProfileType
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "single cpu type",
			input:    []string{"cpu"},
			expected: []pyroscope.ProfileType{pyroscope.ProfileCPU},
		},
		{
			name:  "all profile types",
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
			name:     "empty slice returns nil",
			input:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProfileTypes(tt.input)
			require.Len(t, result, len(tt.expected))
			for i, pt := range tt.expected {
				assert.Equal(t, pt, result[i], "index %d", i)
			}
		})
	}
}

func TestProfileTypeNames(t *testing.T) {
	tests := []struct {
		name      string
		types     []pyroscope.ProfileType
		wantLen   int
		wantNames []string
	}{
		{
			name:      "two profile types",
			types:     []pyroscope.ProfileType{pyroscope.ProfileCPU, pyroscope.ProfileGoroutines},
			wantLen:   2,
			wantNames: []string{"cpu", "goroutines"},
		},
		{
			name:      "nil types returns empty",
			types:     nil,
			wantLen:   0,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := profileTypeNames(tt.types)
			assert.Len(t, names, tt.wantLen)
			if tt.wantNames != nil {
				for i, want := range tt.wantNames {
					assert.Equal(t, want, names[i], "index %d", i)
				}
			}
		})
	}
}

func TestPyroscopeLogger(t *testing.T) {
	t.Run("log adapter does not panic", func(t *testing.T) {
		l := pyroscopeLogger{}
		l.Infof("test info %d", 1)
		l.Debugf("test debug %s", "arg")
		l.Errorf("test error %v", "arg")
	})
}
