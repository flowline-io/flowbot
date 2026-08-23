package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify"
)

func TestDeployNotifyPayload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		user     string
		repo     string
		build    int
		droneURL string
		want     map[string]any
	}{
		{
			name:     "includes summary title and drone url",
			user:     "alice",
			repo:     "deploy",
			build:    42,
			droneURL: "https://drone.example",
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "alice/deploy #42",
				notify.PayloadKeyURL:     "https://drone.example",
			},
		},
		{
			name:  "omits url when drone url is empty",
			user:  "bob",
			repo:  "deploy",
			build: 1,
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "bob/deploy #1",
			},
		},
		{
			name:     "formats zero build number",
			user:     "carol",
			repo:     "deploy",
			build:    0,
			droneURL: "https://ci.local",
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "carol/deploy #0",
				notify.PayloadKeyURL:     "https://ci.local",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deployNotifyPayload(tt.user, tt.repo, tt.build, tt.droneURL)
			require.Equal(t, tt.want, got)
			_, hasURL := got[notify.PayloadKeyURL]
			assert.Equal(t, tt.droneURL != "", hasURL)
		})
	}
}
