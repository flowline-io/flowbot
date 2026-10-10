package homelab

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDigestRuntime struct {
	NoopRuntime
	local       map[string]string
	remote      map[string]string
	localErr    map[string]error
	remoteErr   map[string]error
	localCalls  []string
	remoteCalls []string
}

func (s *stubDigestRuntime) ImageRepoDigest(_ context.Context, app App, svc ComposeService) (string, error) {
	key := app.Name + "/" + svc.Name
	s.localCalls = append(s.localCalls, key)
	if err := s.localErr[key]; err != nil {
		return "", err
	}
	return s.local[key], nil
}

func (s *stubDigestRuntime) RemoteManifestDigest(_ context.Context, app App, imageRef string) (string, error) {
	s.remoteCalls = append(s.remoteCalls, app.Name+":"+imageRef)
	if err := s.remoteErr[imageRef]; err != nil {
		return "", err
	}
	return s.remote[imageRef], nil
}

func TestImageCheckInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mode     RuntimeMode
		interval string
		want     time.Duration
		ok       bool
	}{
		{name: "docker default", mode: RuntimeModeDockerSocket, interval: "", want: DefaultImageCheckInterval, ok: true},
		{name: "ssh custom", mode: RuntimeModeSSH, interval: "1h", want: time.Hour, ok: true},
		{name: "zero disables", mode: RuntimeModeDockerSocket, interval: "0s", ok: false},
		{name: "bare zero disables", mode: RuntimeModeDockerSocket, interval: "0", ok: false},
		{name: "negative disables", mode: RuntimeModeDockerSocket, interval: "-1h", ok: false},
		{name: "none disables", mode: RuntimeModeNone, interval: "", ok: false},
		{name: "invalid disables", mode: RuntimeModeDockerSocket, interval: "nope", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ImageCheckInterval(tt.mode, tt.interval)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckImageUpdates(t *testing.T) {
	t.Parallel()
	apps := []App{
		{
			Name: "karakeep",
			Labels: map[string]string{
				LabelCapability: "karakeep",
			},
			Capabilities: []AppCapability{
				{Capability: "archive"},
			},
			Services: []ComposeService{
				{Name: "web", Image: "ghcr.io/karakeep/karakeep:latest"},
				{Name: "db", Image: "postgres:16", Labels: map[string]string{LabelImageWatch: "false"}},
				{Name: "pinned", Image: "nginx@sha256:dead"},
			},
		},
		{
			Name: "other",
			Capabilities: []AppCapability{
				{Capability: "archive"},
			},
			Services: []ComposeService{
				{Name: "cache", Image: "redis:7"},
			},
		},
	}

	tests := []struct {
		name      string
		rt        *stubDigestRuntime
		want      []ImageUpdate
		wantLocal []string
	}{
		{
			name: "emits mismatch and skips excluded and equal",
			rt: &stubDigestRuntime{
				local: map[string]string{
					"karakeep/web": "sha256:old",
					"other/cache":  "sha256:same",
				},
				remote: map[string]string{
					"ghcr.io/karakeep/karakeep:latest": "sha256:new",
					"redis:7":                          "sha256:same",
				},
			},
			want: []ImageUpdate{
				{
					AppName:       "karakeep",
					Service:       "web",
					Image:         "ghcr.io/karakeep/karakeep:latest",
					Tag:           "latest",
					CurrentDigest: "sha256:old",
					RemoteDigest:  "sha256:new",
					Capability:    "karakeep",
				},
			},
			wantLocal: []string{"karakeep/web", "other/cache"},
		},
		{
			name: "local error skips without remote call for that service",
			rt: &stubDigestRuntime{
				localErr: map[string]error{"karakeep/web": errors.New("inspect failed")},
				local:    map[string]string{"other/cache": "sha256:a"},
				remote:   map[string]string{"redis:7": "sha256:b"},
			},
			want: []ImageUpdate{
				{
					AppName:       "other",
					Service:       "cache",
					Image:         "redis:7",
					Tag:           "7",
					CurrentDigest: "sha256:a",
					RemoteDigest:  "sha256:b",
				},
			},
			wantLocal: []string{"karakeep/web", "other/cache"},
		},
		{
			name: "empty local repo digest skips remote",
			rt: &stubDigestRuntime{
				local:  map[string]string{"karakeep/web": "", "other/cache": ""},
				remote: map[string]string{"ghcr.io/karakeep/karakeep:latest": "sha256:new"},
			},
			want:      nil,
			wantLocal: []string{"karakeep/web", "other/cache"},
		},
		{
			name: "remote error skips that service",
			rt: &stubDigestRuntime{
				local:     map[string]string{"karakeep/web": "sha256:old", "other/cache": "sha256:a"},
				remoteErr: map[string]error{"ghcr.io/karakeep/karakeep:latest": errors.New("429")},
				remote:    map[string]string{"redis:7": "sha256:b"},
			},
			want: []ImageUpdate{
				{
					AppName:       "other",
					Service:       "cache",
					Image:         "redis:7",
					Tag:           "7",
					CurrentDigest: "sha256:a",
					RemoteDigest:  "sha256:b",
				},
			},
			wantLocal: []string{"karakeep/web", "other/cache"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CheckImageUpdates(t.Context(), tt.rt, apps)
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				require.Equal(t, tt.want, got)
			}
			assert.Equal(t, tt.wantLocal, tt.rt.localCalls)
		})
	}
}

func TestCheckImageUpdatesNilRuntime(t *testing.T) {
	t.Parallel()
	assert.Nil(t, CheckImageUpdates(t.Context(), nil, []App{{Name: "x"}}))
}
