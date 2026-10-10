package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchImageRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		image   string
		wantRef string
		wantTag string
		ok      bool
	}{
		{name: "tagged image", image: "nginx:1.27", wantRef: "nginx:1.27", wantTag: "1.27", ok: true},
		{name: "implicit latest", image: "nginx", wantRef: "nginx:latest", wantTag: "latest", ok: true},
		{name: "digest pinned skipped", image: "nginx@sha256:abc", ok: false},
		{name: "empty skipped", image: "", ok: false},
		{name: "registry port untagged", image: "registry.example.com:5000/app", wantRef: "registry.example.com:5000/app:latest", wantTag: "latest", ok: true},
		{name: "registry port tagged", image: "registry.example.com:5000/app:2", wantRef: "registry.example.com:5000/app:2", wantTag: "2", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref, tag, ok := WatchImageRef(tt.image)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.wantRef, ref)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestServiceWatchesImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		svc  ComposeService
		want bool
	}{
		{name: "tagged service is watched", svc: ComposeService{Name: "web", Image: "nginx:latest"}, want: true},
		{name: "digest pinned is not watched", svc: ComposeService{Image: "nginx@sha256:abc"}, want: false},
		{name: "watch false excludes", svc: ComposeService{Image: "nginx:latest", Labels: map[string]string{LabelImageWatch: "false"}}, want: false},
		{name: "watch FALSE excludes", svc: ComposeService{Image: "nginx:latest", Labels: map[string]string{LabelImageWatch: "FALSE"}}, want: false},
		{name: "watch true still watched", svc: ComposeService{Image: "nginx:latest", Labels: map[string]string{LabelImageWatch: "true"}}, want: true},
		{name: "empty image not watched", svc: ComposeService{Name: "web"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ServiceWatchesImage(tt.svc))
		})
	}
}

func TestImageUpdateIdempotencyKey(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "karakeep/web/sha256:abc", ImageUpdateIdempotencyKey("karakeep", "web", "sha256:abc"))
	assert.Equal(t, "karakeep/web", ImageUpdateEntityID("karakeep", "web"))
}
