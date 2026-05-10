package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "IssueWebhookID should equal issue", got: IssueWebhookID, expected: "issue"},
		{name: "RepoWebhookID should equal repo", got: RepoWebhookID, expected: "repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.got)
		})
	}
}

func TestWebhookRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 2 webhook rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, webhookRules, 2)
			},
		},
		{
			name: "should contain expected webhook IDs",
			test: func(t *testing.T) {
				t.Parallel()
				ids := make(map[string]bool)
				for _, r := range webhookRules {
					ids[r.Id] = true
				}

				assert.True(t, ids[IssueWebhookID])
				assert.True(t, ids[RepoWebhookID])
			},
		},
		{
			name: "all webhooks should have Secret=true",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range webhookRules {
					assert.True(t, r.Secret, "webhook %q should have Secret=true", r.Id)
				}
			},
		},
		{
			name: "all webhooks should have non-nil handlers",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range webhookRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
