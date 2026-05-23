package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
