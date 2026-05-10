package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentList_FillUrlBase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		list    DocumentList
		urlBase map[string]string
		want    []string
	}{
		{
			name: "fills URLs from base map",
			list: DocumentList{
				{Source: "github", Url: "/repos/foo"},
				{Source: "gitlab", Url: "/projects/1"},
				{Source: "unknown", Url: "/path"},
			},
			urlBase: map[string]string{
				"github": "https://github.com",
				"gitlab": "https://gitlab.com",
			},
			want: []string{
				"https://github.com/repos/foo",
				"https://gitlab.com/projects/1",
				"/path",
			},
		},
		{
			name:    "empty list",
			list:    nil,
			urlBase: map[string]string{"x": "y"},
			want:    nil,
		},
		{
			name: "no matching base URL",
			list: DocumentList{
				{Source: "bitbucket", Url: "/repo"},
			},
			urlBase: map[string]string{"github": "https://github.com"},
			want:    []string{"/repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.list.FillUrlBase(tt.urlBase)
			if tt.want == nil {
				assert.Nil(t, tt.list)
			} else {
				urls := make([]string, len(tt.list))
				for i, d := range tt.list {
					urls[i] = d.Url
				}
				assert.Equal(t, tt.want, urls)
			}
		})
	}
}

func TestRulesetTypeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		constant RulesetType
		want     RulesetType
	}{
		{
			name:     "ActionRule",
			constant: ActionRule,
			want:     RulesetType("action"),
		},
		{
			name:     "CommandRule",
			constant: CommandRule,
			want:     RulesetType("command"),
		},
		{
			name:     "CronRule",
			constant: CronRule,
			want:     RulesetType("cron"),
		},
		{
			name:     "EventRule",
			constant: EventRule,
			want:     RulesetType("event"),
		},
		{
			name:     "FormRule",
			constant: FormRule,
			want:     RulesetType("form"),
		},
		{
			name:     "PageRule",
			constant: PageRule,
			want:     RulesetType("page"),
		},
		{
			name:     "TriggerRule",
			constant: TriggerRule,
			want:     RulesetType("trigger"),
		},
		{
			name:     "WebhookRule",
			constant: WebhookRule,
			want:     RulesetType("webhook"),
		},
		{
			name:     "WebserviceRule",
			constant: WebserviceRule,
			want:     RulesetType("webservice"),
		},
		{
			name:     "WorkflowRule",
			constant: WorkflowRule,
			want:     RulesetType("workflow"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.constant)
		})
	}
}
