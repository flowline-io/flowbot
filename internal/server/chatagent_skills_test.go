package server

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSkillTool_Execute(t *testing.T) {
	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testAgentSkills = map[string]*gen.AgentSkill{
		"homelab-bookmark": {
			Flag:        "homelab-bookmark",
			Name:        "homelab-bookmark",
			Description: "Bookmark skill",
			Content:     "# Bookmark\nUse flowbot bookmark list",
			Enabled:     true,
		},
		"hidden": {
			Flag:                   "hidden",
			Name:                   "hidden",
			Description:            "Hidden skill",
			Content:                "secret",
			Enabled:                true,
			DisableModelInvocation: true,
		},
	}
	testAgentSkillFiles = map[string]map[string]*gen.AgentSkillFile{
		"homelab-bookmark": {
			"reference.md": {
				SkillFlag: "homelab-bookmark",
				Path:      "reference.md",
				Content:   "Reference body",
			},
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		testAgentSkills = map[string]*gen.AgentSkill{}
		testAgentSkillFiles = map[string]map[string]*gen.AgentSkillFile{}
	})

	tool := chatagent.ReadSkillTool{}
	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		wantText  string
	}{
		{name: "loads skill", args: map[string]any{"name": "homelab-bookmark"}, wantText: "Use flowbot bookmark list"},
		{name: "loads skill file", args: map[string]any{"name": "homelab-bookmark", "path": "reference.md"}, wantText: "Reference body"},
		{name: "missing skill file", args: map[string]any{"name": "homelab-bookmark", "path": "missing.md"}, wantError: true},
		{name: "missing skill", args: map[string]any{"name": "missing"}, wantError: true},
		{name: "hidden skill", args: map[string]any{"name": "hidden"}, wantError: true},
		{name: "empty name", args: map[string]any{"name": "  "}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				part, ok := result.Parts[0].(msg.TextPart)
				require.True(t, ok)
				assert.Contains(t, part.Text, tt.wantText)
			}
		})
	}
}

func TestLoadSkillsFromStore(t *testing.T) {
	origDB := store.Database
	store.Database = &testStoreAdapter{}
	testAgentSkills = map[string]*gen.AgentSkill{
		"visible": {
			Flag:                   "visible",
			Name:                   "visible",
			Description:            "Visible skill",
			Enabled:                true,
			DisableModelInvocation: false,
		},
		"hidden": {
			Flag:                   "hidden",
			Name:                   "hidden",
			Description:            "Hidden skill",
			Enabled:                true,
			DisableModelInvocation: true,
		},
	}
	t.Cleanup(func() {
		store.Database = origDB
		testAgentSkills = map[string]*gen.AgentSkill{}
		testAgentSkillFiles = map[string]map[string]*gen.AgentSkillFile{}
	})

	skills, err := chatagent.LoadSkillsFromStore(context.Background())
	require.NoError(t, err)
	require.Len(t, skills, 2)
}
