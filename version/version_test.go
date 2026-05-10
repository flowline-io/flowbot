package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildstamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "Buildstamp is not empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, Buildstamp)
		})
	}
}

func TestBuildtags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "Buildtags is not empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, Buildtags)
		})
	}
}
