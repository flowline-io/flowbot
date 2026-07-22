package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomeQuickLinksPendingApprovals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		pendingApprovals int
		wantAgentsDetail string
		wantAgentsBadge  int
	}{
		{
			name:             "idle agents detail unchanged",
			pendingApprovals: 0,
			wantAgentsDetail: "Chat and orchestrate",
			wantAgentsBadge:  0,
		},
		{
			name:             "one pending updates agents detail",
			pendingApprovals: 1,
			wantAgentsDetail: "1 awaiting approval",
			wantAgentsBadge:  1,
		},
		{
			name:             "multiple pending updates agents detail",
			pendingApprovals: 4,
			wantAgentsDetail: "4 awaiting approval",
			wantAgentsBadge:  4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			links := HomeQuickLinks(tt.pendingApprovals)
			assert.GreaterOrEqual(t, len(links), 1)
			assert.Equal(t, "Agents", links[0].Title)
			assert.Equal(t, tt.wantAgentsDetail, links[0].Detail)
			assert.Equal(t, tt.wantAgentsBadge, links[0].Badge)
			assert.Equal(t, "home-link-agents", links[0].TestID)
		})
	}
}
