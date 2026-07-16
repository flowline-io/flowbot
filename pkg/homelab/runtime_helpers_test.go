package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseComposePSStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   AppStatus
	}{
		{name: "empty output is stopped", output: "", want: AppStatusStopped},
		{name: "empty array is stopped", output: "[]", want: AppStatusStopped},
		{name: "null is stopped", output: "null", want: AppStatusStopped},
		{name: "all running", output: `[{"State":"running"},{"State":"running"}]`, want: AppStatusRunning},
		{name: "mixed running and exited", output: `[{"State":"running"},{"State":"exited"}]`, want: AppStatusPartial},
		{name: "all exited", output: `[{"State":"exited"},{"State":"dead"}]`, want: AppStatusStopped},
		{name: "invalid json is unknown", output: `{not json`, want: AppStatusUnknown},
		{name: "unknown state is unknown", output: `[{"State":"created"}]`, want: AppStatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseComposePSStatus(tt.output))
		})
	}
}

func TestShellQuote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple string", in: "hello", want: "'hello'"},
		{name: "embedded single quote escaped", in: "it's fine", want: "'it'\"'\"'s fine'"},
		{name: "spaces preserved", in: "a b c", want: "'a b c'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shellQuote(tt.in))
		})
	}
}

func TestAllowlistSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		values []string
		check  func(t *testing.T, got map[string]bool)
	}{
		{
			name:   "empty slice",
			values: nil,
			check: func(t *testing.T, got map[string]bool) {
				t.Helper()
				assert.Empty(t, got)
			},
		},
		{
			name:   "skips empty strings",
			values: []string{"app1", "", "app2"},
			check: func(t *testing.T, got map[string]bool) {
				t.Helper()
				assert.True(t, got["app1"])
				assert.True(t, got["app2"])
				assert.Len(t, got, 2)
			},
		},
		{
			name:   "deduplicates by key presence",
			values: []string{"nginx", "redis"},
			check: func(t *testing.T, got map[string]bool) {
				t.Helper()
				assert.True(t, got["nginx"])
				assert.True(t, got["redis"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, allowlistSet(tt.values))
		})
	}
}

func TestIsInside(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{name: "same path", root: "/apps", path: "/apps", want: true},
		{name: "child path", root: "/apps", path: "/apps/nginx", want: true},
		{name: "parent path rejected", root: "/apps/nginx", path: "/apps", want: false},
		{name: "sibling path rejected", root: "/apps/a", path: "/apps/b", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isInside(tt.root, tt.path))
		})
	}
}

func TestDefaultProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		protocol string
		want     string
	}{
		{name: "empty defaults to tcp", protocol: "", want: "tcp"},
		{name: "udp preserved", protocol: "udp", want: "udp"},
		{name: "tcp preserved", protocol: "tcp", want: "tcp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, defaultProtocol(tt.protocol))
		})
	}
}

func TestStringMapValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value map[string]any
		key   string
		want  string
	}{
		{name: "missing key", value: map[string]any{}, key: "port", want: ""},
		{name: "nil value", value: map[string]any{"port": nil}, key: "port", want: ""},
		{name: "int value", value: map[string]any{"port": 8080}, key: "port", want: "8080"},
		{name: "string via default branch", value: map[string]any{"host": "localhost"}, key: "host", want: "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stringMapValue(tt.value, tt.key))
		})
	}
}
