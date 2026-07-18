package partials_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestCapsLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		caps []homelab.AppCapability
		want string
	}{
		{name: "empty", caps: nil, want: ""},
		{
			name: "single",
			caps: []homelab.AppCapability{{Capability: "gitea"}},
			want: "gitea",
		},
		{
			name: "multiple joined with middle dot",
			caps: []homelab.AppCapability{
				{Capability: "gitea"},
				{Capability: "notify"},
				{Capability: ""},
			},
			want: "gitea · notify",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := partials.CapsLabel(tt.caps); got != tt.want {
				t.Fatalf("CapsLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
