package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifestDigest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "plain digest", output: "sha256:abcdef0123456789", want: "sha256:abcdef0123456789"},
		{name: "trailing newline", output: "sha256:abcdef0123456789\n", want: "sha256:abcdef0123456789"},
		{name: "name at digest", output: "nginx@sha256:abcdef0123456789", want: "sha256:abcdef0123456789"},
		{name: "warning then digest", output: "WARNING: foo\nsha256:abcdef0123456789\n", want: "sha256:abcdef0123456789"},
		{name: "empty", output: "", want: ""},
		{name: "noise only", output: "error: not found", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseManifestDigest(tt.output))
		})
	}
}

func TestPickRepoDigest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		digests []string
		image   string
		want    string
	}{
		{
			name:    "matches canonical name",
			digests: []string{"docker.io/library/nginx@sha256:aaa", "redis@sha256:bbb"},
			image:   "nginx:latest",
			want:    "sha256:aaa",
		},
		{
			name:    "short name matches library path",
			digests: []string{"nginx@sha256:ccc"},
			image:   "docker.io/library/nginx:1.27",
			want:    "sha256:ccc",
		},
		{
			name:    "unrelated name is skipped",
			digests: []string{"other@sha256:ddd"},
			image:   "nginx:latest",
			want:    "",
		},
		{
			name:    "empty list",
			digests: nil,
			image:   "nginx:latest",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, pickRepoDigest(tt.digests, tt.image))
		})
	}
}

func TestRepoDigestFromInspect(t *testing.T) {
	t.Parallel()
	got, err := repoDigestFromInspect(`["nginx@sha256:abc123"]`, "nginx:latest")
	require.NoError(t, err)
	assert.Equal(t, "sha256:abc123", got)

	got, err = repoDigestFromInspect("null", "nginx:latest")
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = repoDigestFromInspect("{not json}", "nginx:latest")
	require.Error(t, err)
}

func TestFirstNonEmptyLine(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "abc", firstNonEmptyLine("\n  abc \n def"))
	assert.Empty(t, firstNonEmptyLine("  \n"))
}
