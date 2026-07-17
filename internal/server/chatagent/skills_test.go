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
				Name:        "karakeep",
				Description: "Manage bookmarks via Flowbot CLI",
				Location:    "skill://karakeep",
			}},
			wantParts: []string{
				"<available_skills>",
				"<name>karakeep</name>",
				"Manage bookmarks via Flowbot CLI",
				"read_skill",
			},
		},
		{
			name: "renders skill files",
			skills: []chatagent.Skill{{
				Name:        "karakeep",
				Description: "Manage bookmarks via Flowbot CLI",
				Location:    "skill://karakeep",
				Files:       []string{"reference.md", "scripts/run.sh"},
			}},
			wantParts: []string{
				"<files>",
				"<file>reference.md</file>",
				"<file>scripts/run.sh</file>",
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

func TestFilterSkillsByNames(t *testing.T) {
	all := []chatagent.Skill{
		{Name: "alpha", Description: "Alpha skill"},
		{Name: "beta", Description: "Beta skill"},
		{Name: "gamma", Description: "Gamma skill"},
		{Name: "karakeep", Description: "Bookmark skill"},
	}

	tests := []struct {
		name      string
		allowlist []string
		wantNames []string
	}{
		{name: "filters matching skills", allowlist: []string{"alpha", "gamma"}, wantNames: []string{"alpha", "gamma"}},
		{name: "empty allowlist returns none", allowlist: nil, wantNames: nil},
		{name: "unknown names ignored", allowlist: []string{"missing"}, wantNames: []string{}},
		{name: "legacy allowlist matches cap id skill", allowlist: []string{"homelab-bookmark"}, wantNames: []string{"karakeep"}},
		{name: "cap id allowlist matches", allowlist: []string{"karakeep"}, wantNames: []string{"karakeep"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatagent.FilterSkillsByNames(all, tt.allowlist)
			if tt.wantNames == nil {
				assert.Nil(t, got)
				return
			}
			names := make([]string, 0, len(got))
			for _, skill := range got {
				names = append(names, skill.Name)
			}
			assert.ElementsMatch(t, tt.wantNames, names)
		})
	}
}

func TestCanonicalSkillName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "legacy bookmark", in: "homelab-bookmark", want: "karakeep"},
		{name: "legacy kanban", in: "homelab-kanban", want: "kanboard"},
		{name: "legacy reader", in: "homelab-reader", want: "miniflux"},
		{name: "already canonical", in: "karakeep", want: "karakeep"},
		{name: "unknown unchanged", in: "custom-skill", want: "custom-skill"},
		{name: "trims space", in: "  homelab-bookmark  ", want: "karakeep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatagent.CanonicalSkillName(tt.in))
		})
	}
}
