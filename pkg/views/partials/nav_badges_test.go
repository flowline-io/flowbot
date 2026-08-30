package partials_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestNavBadgeFragmentsCopyToReplicaSlots(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	tests := []struct {
		name   string
		render func() (string, error)
		wants  []string
	}{
		{
			name: "inbox copies to mobile slot",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.InboxCountBadgeNav(3).Render(ctx, &buf)
				return buf.String(), err
			},
			wants: []string{
				`data-testid="inbox-count-badge"`,
				`hx-swap-oob="innerHTML:#` + partials.NavMobileInboxBadgeID + `"`,
			},
		},
		{
			name: "approval copies to mobile slot",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.ApprovalCountBadgeNav(2).Render(ctx, &buf)
				return buf.String(), err
			},
			wants: []string{
				`hx-swap-oob="innerHTML:#` + partials.NavMobileApprovalBadgeID + `"`,
			},
		},
		{
			name: "approval copies to agents menu slot",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.ApprovalCountBadgeNav(2).Render(ctx, &buf)
				return buf.String(), err
			},
			wants: []string{
				`hx-swap-oob="innerHTML:#` + partials.NavAgentsApprovalBadgeID + `"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html, err := tt.render()
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			for _, want := range tt.wants {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in badge fragment, got %q", want, html)
				}
			}
		})
	}
}
