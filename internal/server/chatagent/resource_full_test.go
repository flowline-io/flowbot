package chatagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveResourceFullOption(t *testing.T) {
	LockAppConfigForTest(t)

	origCfg := config.App.ChatAgent
	origDB := store.Database
	root := t.TempDir()
	large := strings.Repeat("x", coding.DefaultMaxOutput+200)
	require.NoError(t, os.WriteFile(filepath.Join(root, "big.txt"), []byte(large), 0o600))
	config.App.ChatAgent = config.ChatAgentConfig{Workspace: root, ChatModel: "gpt-test"}
	store.Database = postgres.NewSQLiteTestAdapter(t)
	sessionID := types.Id()
	require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
		Flag:  sessionID,
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
	}))
	t.Cleanup(func() {
		config.App.ChatAgent = origCfg
		store.Database = origDB
	})

	tests := []struct {
		name          string
		sessionID     string
		opts          ResolveResourceOptions
		wantErr       bool
		wantTruncated bool
		wantSuffix    string
		wantFullLen   bool
	}{
		{
			name:          "default truncates large file",
			sessionID:     sessionID,
			opts:          ResolveResourceOptions{},
			wantTruncated: true,
			wantSuffix:    "\n...(output truncated)",
		},
		{
			name:          "full skips truncate output",
			sessionID:     sessionID,
			opts:          ResolveResourceOptions{Full: true},
			wantTruncated: false,
			wantFullLen:   true,
		},
		{
			name:      "rejects empty session id",
			sessionID: "",
			opts:      ResolveResourceOptions{Full: true},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			got, err := svc.ResolveResourceWithOptions(context.Background(), tt.sessionID, "file://big.txt", tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTruncated, got.Truncated)
			if tt.wantSuffix != "" {
				assert.True(t, strings.HasSuffix(got.Content, tt.wantSuffix))
			}
			if tt.wantFullLen {
				assert.Equal(t, large, got.Content)
			}
		})
	}
}

func TestLimitResourcePreviewContent(t *testing.T) {
	t.Parallel()
	overCap := strings.Repeat("a", ResourcePreviewMaxBytes+10)
	tests := []struct {
		name          string
		raw           string
		full          bool
		wantTruncated bool
		wantLenMax    int
	}{
		{
			name:          "preview truncates with workspace limit marker path",
			raw:           strings.Repeat("b", coding.DefaultMaxOutput+5),
			full:          false,
			wantTruncated: true,
			wantLenMax:    coding.DefaultMaxOutput + len("\n...(output truncated)"),
		},
		{
			name:          "full under hard cap returns intact",
			raw:           "short",
			full:          true,
			wantTruncated: false,
			wantLenMax:    5,
		},
		{
			name:          "full over hard cap truncates",
			raw:           overCap,
			full:          true,
			wantTruncated: true,
			wantLenMax:    ResourcePreviewMaxBytes + len("\n...(output truncated)"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content, truncated := LimitResourcePreviewContent(tt.raw, tt.full)
			assert.Equal(t, tt.wantTruncated, truncated)
			assert.LessOrEqual(t, len(content), tt.wantLenMax)
		})
	}
}
