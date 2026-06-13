package chatagent_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildContextUsageReport(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# rules"), 0o644))

	prev := config.App
	t.Cleanup(func() { config.App = prev })

	config.App = config.Type{
		Models: []config.Model{{
			ModelNames:     []string{"test-model"},
			ContextWindows: map[string]int{"test-model": 100000},
		}},
		ChatAgent: config.ChatAgentConfig{
			ChatModel: "test-model",
			Workspace: root,
			Compaction: config.CompactionConfig{
				Enabled:       true,
				ReserveTokens: 10000,
			},
		},
	}

	tests := []struct {
		name           string
		sessionID      string
		wantModel      string
		wantWindow     int
		wantCategories []string
		wantMinTotal   int
	}{
		{
			name:           "empty session still reports prompt overhead",
			sessionID:      "",
			wantModel:      "test-model",
			wantWindow:     100000,
			wantCategories: []string{"system_prompt", "system_tools", "skills", "messages", "free_space", "autocompact_buffer"},
			wantMinTotal:   100,
		},
		{
			name:           "unknown session id skips message tokens",
			sessionID:      "missing-session",
			wantModel:      "test-model",
			wantWindow:     100000,
			wantCategories: []string{"system_prompt", "system_tools", "skills", "messages", "free_space", "autocompact_buffer"},
			wantMinTotal:   100,
		},
		{
			name:           "reports compaction reserve",
			sessionID:      "",
			wantModel:      "test-model",
			wantWindow:     100000,
			wantCategories: []string{"autocompact_buffer"},
			wantMinTotal:   100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := chatagent.BuildContextUsageReport(context.Background(), tt.sessionID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantModel, report.Model)
			assert.Equal(t, tt.wantWindow, report.ContextWindow)
			assert.GreaterOrEqual(t, report.TotalTokens, tt.wantMinTotal)

			gotIDs := make([]string, 0, len(report.Categories))
			for _, cat := range report.Categories {
				gotIDs = append(gotIDs, cat.ID)
			}
			for _, id := range tt.wantCategories {
				assert.Contains(t, gotIDs, id)
			}

			var bufferTokens int
			for _, cat := range report.Categories {
				if cat.ID == "autocompact_buffer" {
					bufferTokens = cat.Tokens
				}
			}
			if tt.name == "reports compaction reserve" {
				assert.Equal(t, 10000, bufferTokens)
			}
		})
	}
}

func TestEstimateTextTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "short text", text: "hello", want: 2},
		{name: "longer text", text: string(make([]byte, 1000)), want: 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatagent.EstimateTextTokens(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}
