package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultExportFilename(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		wantSub   string
	}{
		{name: "uses session id", sessionID: "sess-123", wantSub: "flowbot-chat-sess-123.json"},
		{name: "sanitizes unsafe chars", sessionID: "sess/123", wantSub: "flowbot-chat-sess123.json"},
		{name: "fallback name", sessionID: "///", wantSub: "flowbot-chat-session.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultExportFilename(tt.sessionID)
			assert.Equal(t, tt.wantSub, got)
		})
	}
}

func TestResolveExportPath(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		sessionID  string
		wantSuffix string
		wantErr    bool
	}{
		{name: "default filename", args: "", sessionID: "abc123", wantSuffix: "flowbot-chat-abc123.json"},
		{name: "custom path", args: "out/chat", sessionID: "abc123", wantSuffix: "out/chat.json"},
		{name: "keeps json extension", args: "out/chat.json", sessionID: "abc123", wantSuffix: "out/chat.json"},
		{name: "ignores placeholder arg", args: "[path]", sessionID: "abc123", wantSuffix: "flowbot-chat-abc123.json"},
		{name: "ignores angle placeholder", args: "<path>", sessionID: "abc123", wantSuffix: "flowbot-chat-abc123.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveExportPath(tt.args, tt.sessionID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(filepath.ToSlash(got), filepath.ToSlash(tt.wantSuffix)))
		})
	}
}

func TestWriteSessionExport(t *testing.T) {
	tests := []struct {
		name   string
		export *client.ChatSessionExport
	}{
		{
			name: "writes full session export",
			export: &client.ChatSessionExport{
				SessionID:  "sess-1",
				UID:        "user-1",
				LeafID:     "e2",
				State:      "active",
				ExportedAt: time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC),
				EntryCount: 2,
				Entries:    []map[string]any{{"id": "e1", "type": "message"}, {"id": "e2", "type": "compaction"}},
			},
		},
		{
			name: "writes empty entries",
			export: &client.ChatSessionExport{
				SessionID:  "sess-2",
				ExportedAt: time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name:   "rejects nil export",
			export: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "export.json")
			err := WriteSessionExport(path, tt.export)
			if tt.export == nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)

			var loaded client.ChatSessionExport
			require.NoError(t, sonic.Unmarshal(data, &loaded))
			assert.Equal(t, tt.export.SessionID, loaded.SessionID)
			assert.Equal(t, tt.export.EntryCount, loaded.EntryCount)
			assert.Len(t, loaded.Entries, len(tt.export.Entries))
		})
	}
}

func TestFormatExportSuccess(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		count int
		want  string
	}{
		{name: "single entry", path: "/tmp/chat.json", count: 1, want: "Exported 1 entries to /tmp/chat.json"},
		{name: "many entries", path: "chat.json", count: 12, want: "Exported 12 entries to chat.json"},
		{name: "empty session", path: "empty.json", count: 0, want: "Exported 0 entries to empty.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatExportSuccess(tt.path, tt.count))
		})
	}
}
