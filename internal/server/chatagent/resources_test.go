package chatagent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResourceURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		wantScheme string
		wantRef    string
		wantErr    bool
	}{
		{name: "plan uri", uri: "plan://abc123", wantScheme: "plan", wantRef: "abc123"},
		{name: "file uri", uri: "file://src/main.go", wantScheme: "file", wantRef: "src/main.go"},
		{name: "empty plan id", uri: "plan://", wantErr: true},
		{name: "unsupported scheme", uri: "skill://demo", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, ref, err := ParseResourceURI(tt.uri)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantScheme, scheme)
			assert.Equal(t, tt.wantRef, ref)
		})
	}
}

func TestResolveFileResource(t *testing.T) {
	origCfg := config.App.ChatAgent
	origDB := store.Database
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "note.txt"), []byte("hello file"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "secrets.env"), []byte("SECRET=1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "bad.bin"), []byte{0xff, 0xfe, 0x00}, 0o600))
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
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{name: "reads text file", uri: "file://note.txt", want: "hello file"},
		{name: "rejects env file", uri: "file://secrets.env", wantErr: true},
		{name: "rejects binary", uri: "file://bad.bin", wantErr: true},
		{name: "rejects escape", uri: "file://../outside.txt", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveResource(context.Background(), sessionID, tt.uri)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Content)
		})
	}
}

func TestExtractResourceURIs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "no links", text: "plain text", want: nil},
		{name: "plan link", text: "see [Plan](plan://abc)", want: []string{"plan://abc"}},
		{name: "dedupe", text: "[a](plan://x) and [b](plan://x)", want: []string{"plan://x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractResourceURIs(tt.text))
		})
	}
}

func TestDerivePlanTitle(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  string
	}{
		{name: "first heading line", reply: "# Deploy plan\nstep 1", want: "Deploy plan"},
		{name: "fallback", reply: "\n\n", want: "Plan"},
		{name: "long line truncated", reply: repeatChar('x', 120), want: repeatChar('x', 80)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derivePlanTitle(tt.reply))
		})
	}
}

func TestAppendPlanLinkFooter(t *testing.T) {
	got := AppendPlanLinkFooter("body", "p1", "Title")
	assert.Contains(t, got, "plan://p1")
	assert.Contains(t, got, "Title")
}

func TestFormatPlanResourceRef(t *testing.T) {
	ref := FormatPlanResourceRef("p1", "Title")
	assert.Equal(t, "plan://p1", ref.URI)
	assert.Equal(t, "plan", ref.Kind)
	assert.Equal(t, "Title", ref.Title)
}

func repeatChar(ch byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
