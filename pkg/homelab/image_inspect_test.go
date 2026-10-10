package homelab

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchImageRepoDigestUsesRunningContainerThenImage(t *testing.T) {
	t.Parallel()
	app := App{Name: "app", Path: "/apps/app"}
	svc := ComposeService{Name: "web", Image: "nginx:latest"}

	tests := []struct {
		name    string
		psOut   string
		psErr   error
		inspect map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "running container digest",
			psOut: "abc123\n",
			inspect: map[string]string{
				"abc123": `["nginx@sha256:running"]`,
			},
			want: "sha256:running",
		},
		{
			name:  "stopped falls back to image inspect",
			psOut: "",
			inspect: map[string]string{
				"nginx:latest": `["nginx@sha256:local"]`,
			},
			want: "sha256:local",
		},
		{
			name:    "image inspect error",
			psErr:   errors.New("not running"),
			inspect: map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compose := func(context.Context, App, ...string) (string, error) {
				return tt.psOut, tt.psErr
			}
			docker := func(_ context.Context, args ...string) (string, error) {
				key := args[len(args)-1]
				out, ok := tt.inspect[key]
				if !ok {
					return "", errors.New("missing")
				}
				return out, nil
			}
			got, err := fetchImageRepoDigest(t.Context(), app, svc, compose, docker, "inspect", "image inspect")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
