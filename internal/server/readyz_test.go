package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHTTPStopping(t *testing.T) {
	tests := []struct {
		name string
		set  bool
		want bool
	}{
		{name: "default not stopping", set: false, want: false},
		{name: "marked stopping", set: true, want: true},
		{name: "cleared after stop", set: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHTTPStopping(tt.set)
			assert.Equal(t, tt.want, isHTTPStopping())
		})
	}
	setHTTPStopping(false)
}

func TestReadinessOK_WhenStopping(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stopping fails even without probing deps"},
		{name: "stopping remains fail closed"},
		{name: "clearing stop restores probe path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHTTPStopping(true)
			assert.False(t, readinessOK(t.Context()))
			if tt.name == "clearing stop restores probe path" {
				setHTTPStopping(false)
				// Without Database/Redis wired, still false — only asserts stop flag cleared.
				assert.False(t, isHTTPStopping())
			}
			setHTTPStopping(false)
		})
	}
}
