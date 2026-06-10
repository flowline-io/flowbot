package chatagent_test

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/stretchr/testify/assert"
)

func TestFormatSkillsForPrompt(t *testing.T) {
	tests := []struct {
		name       string
		skills     []chatagent.Skill
		wantParts  []string
		wantAbsent []string
	}{
		{
			name: "renders visible skills",
			skills: []chatagent.Skill{{
				Name:        "homelab-bookmark",
				Description: "Manage bookmarks via Flowbot CLI",
				Location:    "skill://homelab-bookmark",
			}},
			wantParts: []string{
				"<available_skills>",
				"<name>homelab-bookmark</name>",
				"Manage bookmarks via Flowbot CLI",
				"read_skill",
			},
		},
		{
			name: "skips disabled model invocation skills",
			skills: []chatagent.Skill{{
				Name:                   "hidden",
				Description:            "Hidden skill",
				DisableModelInvocation: true,
			}},
			wantAbsent: []string{"<available_skills>"},
		},
		{
			name:       "empty skills",
			skills:     nil,
			wantAbsent: []string{"<available_skills>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatagent.FormatSkillsForPrompt(tt.skills)
			for _, part := range tt.wantParts {
				assert.Contains(t, got, part)
			}
			for _, part := range tt.wantAbsent {
				assert.NotContains(t, got, part)
			}
		})
	}
}
