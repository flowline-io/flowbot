package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewKey verifies that NewKey produces the expected string representation
// for various key combinations.
func TestNewKey(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		entity     string
		identifier string
		want       string
	}{
		{
			name:       "online agent key",
			prefix:     "online",
			entity:     "agent",
			identifier: "host123",
			want:       "online:agent:host123",
		},
		{
			name:       "chat session key",
			prefix:     "chat",
			entity:     "session",
			identifier: "user456",
			want:       "chat:session:user456",
		},
		{
			name:       "notify throttle with compound identifier",
			prefix:     "notify",
			entity:     "throttle",
			identifier: "rule1:eventA:slack",
			want:       "notify:throttle:rule1:eventA:slack",
		},
		{
			name:       "cron filter key",
			prefix:     "cron",
			entity:     "filter",
			identifier: "job1:uid123",
			want:       "cron:filter:job1:uid123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := NewKey(tt.prefix, tt.entity, tt.identifier)
			require.Equal(t, tt.want, k.String())
		})
	}
}

// TestKeyString verifies the String() method output for edge cases.
func TestKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  Key
		want string
	}{
		{
			name: "key with all fields",
			key:  Key{Prefix: "a", Entity: "b", Identifier: "c"},
			want: "a:b:c",
		},
		{
			name: "key with empty identifier",
			key:  Key{Prefix: "a", Entity: "b", Identifier: ""},
			want: "a:b:",
		},
		{
			name: "key with empty entity",
			key:  Key{Prefix: "a", Entity: "", Identifier: "c"},
			want: "a::c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.key.String())
		})
	}
}
