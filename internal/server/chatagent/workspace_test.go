package chatagent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeWorkspaceRel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "empty is root", in: "  ", want: ""},
		{name: "single segment", in: "flowbot", want: "flowbot"},
		{name: "rejects parent", in: "..", wantErr: true},
		{name: "rejects nested", in: "a/b", wantErr: true},
		{name: "rejects hidden", in: ".git", wantErr: true},
		{name: "rejects abs unix", in: "/tmp/x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeWorkspaceRel(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, types.ErrInvalidArgument)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListWorkspaceChoicesSkipsDotDirs(t *testing.T) {
	LockAppConfigForTest(t)
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o750))
	require.NoError(t, os.Mkdir(filepath.Join(root, "zeta"), 0o750))
	require.NoError(t, os.Mkdir(filepath.Join(root, ".hidden"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o640))

	orig := config.App.ChatAgent.Workspace
	t.Cleanup(func() { config.App.ChatAgent.Workspace = orig })
	config.App.ChatAgent.Workspace = root

	got := ListWorkspaceChoices()
	require.Len(t, got, 3)
	assert.Empty(t, got[0].Value)
	assert.Equal(t, filepath.Base(root), got[0].Label)
	assert.Equal(t, "alpha", got[1].Value)
	assert.Equal(t, "zeta", got[2].Value)
}

func TestResolveWorkspaceMissingDir(t *testing.T) {
	LockAppConfigForTest(t)
	root := t.TempDir()
	orig := config.App.ChatAgent.Workspace
	t.Cleanup(func() { config.App.ChatAgent.Workspace = orig })
	config.App.ChatAgent.Workspace = root

	_, err := ResolveWorkspace("gone")
	require.ErrorIs(t, err, types.ErrInvalidArgument)

	require.NoError(t, os.WriteFile(filepath.Join(root, "notdir"), []byte("x"), 0o640))
	_, err = ResolveWorkspace("notdir")
	require.ErrorIs(t, err, types.ErrInvalidArgument)

	config.App.ChatAgent.Workspace = filepath.Join(root, "missing-root")
	_, err = ResolveWorkspace("")
	require.Error(t, err)
	assert.NotErrorIs(t, err, types.ErrInvalidArgument)
}

func TestApplyCreateWorkspaceAndForSession(t *testing.T) {
	LockAppConfigForTest(t)
	installSQLiteTestDatabase(t)
	root := t.TempDir()
	sub := filepath.Join(root, "proj")
	require.NoError(t, os.Mkdir(sub, 0o750))
	orig := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = orig })
	config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-test", Workspace: root}

	ctx := context.Background()
	require.NoError(t, store.ChatStoreFromDB().CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-ws", UID: "u1", State: int(schema.ChatSessionActive),
	}))
	require.NoError(t, ApplyCreateWorkspace(ctx, "sess-ws", "proj"))

	sess, err := store.ChatStoreFromDB().GetChatSession(ctx, "sess-ws")
	require.NoError(t, err)
	assert.Equal(t, "proj", sess.Workspace)

	ws, err := WorkspaceForSession(ctx, "sess-ws")
	require.NoError(t, err)
	want, err := filepath.Abs(sub)
	require.NoError(t, err)
	assert.Equal(t, want, ws.Root)

	require.NoError(t, os.RemoveAll(sub))
	_, err = WorkspaceForSession(ctx, "sess-ws")
	require.Error(t, err)
}

func TestRejectSettingsWorkspaceField(t *testing.T) {
	t.Parallel()
	require.NoError(t, RejectSettingsWorkspaceField([]byte(`{"model":"x"}`)))
	err := RejectSettingsWorkspaceField([]byte(`{"workspace":"foo"}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidArgument)
}

func TestDetectExternalPathUsesSessionRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	foo := filepath.Join(root, "foo")
	bar := filepath.Join(root, "bar")
	require.NoError(t, os.Mkdir(foo, 0o750))
	require.NoError(t, os.Mkdir(bar, 0o750))

	event := hooks.ToolCallEvent{
		ToolCall: msg.ToolCallPart{Name: permission.ToolReadFile},
		Args:     map[string]any{"path": "ok.go"},
	}
	assert.False(t, detectExternalPath(event, foo))

	event.Args = map[string]any{"path": filepath.Join("..", "bar", "x.go")}
	assert.True(t, detectExternalPath(event, foo))
}
