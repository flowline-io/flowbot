package chatagent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSystemPrompt(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Project rules\nUse TDD."), 0o644))

	config.App.Flowbot.Language = "Chinese"

	tests := []struct {
		name       string
		options    chatagent.BuildSystemPromptOptions
		wantParts  []string
		wantAbsent []string
	}{
		{
			name: "default prompt includes tools and workspace",
			options: chatagent.BuildSystemPromptOptions{
				CWD: root,
			},
			wantParts: []string{
				"agent harness",
				"Agent harness:",
				"Observe-Think-Act loop",
				"Harness behavior you should expect:",
				"Available tools:",
				"- read_file:",
				"- run_terminal:",
				"Current working directory:",
				strings.ReplaceAll(root, "\\", "/"),
				"Response language: Chinese",
				"<project_context>",
				"Project rules",
			},
		},
		{
			name: "custom prompt preserves append and context",
			options: chatagent.BuildSystemPromptOptions{
				CustomPrompt:       "You are a specialist.",
				AppendSystemPrompt: "Always cite file paths.",
				CWD:                root,
			},
			wantParts: []string{
				"You are a specialist.",
				"Always cite file paths.",
				"Project rules",
			},
			wantAbsent: []string{"Available tools:"},
		},
		{
			name: "extra guidelines are deduplicated",
			options: chatagent.BuildSystemPromptOptions{
				CWD: root,
				PromptGuidelines: []string{
					"Run tests after edits",
					"Be concise in your responses",
				},
			},
			wantParts: []string{
				"- Run tests after edits",
				"- Be concise in your responses",
			},
		},
		{
			name: "skills section rendered when read_skill active",
			options: chatagent.BuildSystemPromptOptions{
				CWD: root,
				Skills: []chatagent.Skill{{
					Name:        "homelab-bookmark",
					Description: "Manage bookmarks",
					Location:    "skill://homelab-bookmark",
				}},
			},
			wantParts: []string{
				"<available_skills>",
				"homelab-bookmark",
				"- read_skill:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatagent.BuildSystemPrompt(tt.options)
			for _, part := range tt.wantParts {
				assert.Contains(t, got, part)
			}
			for _, part := range tt.wantAbsent {
				assert.NotContains(t, got, part)
			}
		})
	}
}

func TestDefaultToolSnippets(t *testing.T) {
	tests := []struct {
		name string
		tool string
	}{
		{name: "run_terminal", tool: "run_terminal"},
		{name: "read_file", tool: "read_file"},
		{name: "web_search", tool: "web_search"},
	}

	snippets := chatagent.DefaultToolSnippets()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, snippets[tt.tool])
		})
	}
}
