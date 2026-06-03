package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImageVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "image_with_tag",
			image:    "gitea/gitea:1.22.3",
			expected: "1.22.3",
		},
		{
			name:     "image_with_alpine_tag",
			image:    "postgres:16-alpine",
			expected: "16-alpine",
		},
		{
			name:     "image_without_tag",
			image:    "nginx",
			expected: "",
		},
		{
			name:     "image_with_digest",
			image:    "nginx@sha256:abc123",
			expected: "",
		},
		{
			name:     "image_with_registry_and_tag",
			image:    "docker.io/library/redis:7.0",
			expected: "7.0",
		},
		{
			name:     "empty_image",
			image:    "",
			expected: "",
		},
		{
			name:     "registry_with_port_no_tag",
			image:    "registry.example.com:5000/imagename",
			expected: "",
		},
		{
			name:     "registry_with_port_and_tag",
			image:    "registry.example.com:5000/imagename:1.0",
			expected: "1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseImageVersion(tt.image)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAppVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		services []ComposeService
		expected string
	}{
		{
			name: "first_service_with_tag",
			services: []ComposeService{
				{Name: "app", Image: "gitea/gitea:1.22.3"},
				{Name: "db", Image: "postgres:16"},
			},
			expected: "1.22.3",
		},
		{
			name: "skip_service_without_tag",
			services: []ComposeService{
				{Name: "app", Image: "nginx"},
				{Name: "db", Image: "postgres:16"},
			},
			expected: "16",
		},
		{
			name: "all_services_without_tag",
			services: []ComposeService{
				{Name: "app", Image: "nginx"},
				{Name: "db", Image: "alpine"},
			},
			expected: "",
		},
		{
			name:     "no_services",
			services: []ComposeService{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := App{Name: tt.name, Services: tt.services}
			got := AppVersion(app)
			assert.Equal(t, tt.expected, got)
		})
	}
}
