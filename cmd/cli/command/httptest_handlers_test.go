package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockCLI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
	require.NoError(t, os.MkdirAll(cfgDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte(srv.URL), 0600))
}

func wireClientFlags(cmd *cobra.Command) {
	cmd.Flags().String("server-url", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().Bool("debug", false, "")
}

func okJSON(data string) string {
	return `{"status":"ok","data":` + data + `}`
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})
	runErr := fn()
	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)
	return buf.String(), runErr
}

func runCommand(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	wireClientFlags(cmd)
	cmd.SetIn(bytes.NewReader(nil))
	cmd.SetContext(context.Background())
	cmd.SetArgs(args)
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	return out
}

func runCommandExpectError(t *testing.T, cmd *cobra.Command, wantSubstr string, args ...string) {
	t.Helper()
	wireClientFlags(cmd)
	cmd.SetIn(bytes.NewReader(nil))
	cmd.SetContext(context.Background())
	cmd.SetArgs(args)
	_, err := captureStdout(t, cmd.Execute)
	require.Error(t, err)
	if wantSubstr != "" {
		assert.Contains(t, err.Error(), wantSubstr)
	}
}

func TestBookmarkCreateRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
		errSubstr  string
	}{
		{
			name: "create bookmark success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/service/karakeep", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"id":"bm-1","title":"Example","content":{"url":"https://example.com"}}`)))
			},
			args:       []string{"--url", "https://example.com"},
			wantSubstr: "Bookmark created",
		},
		{
			name: "create bookmark api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"create failed"}`))
			},
			args:      []string{"--url", "https://example.com"},
			wantErr:   true,
			errSubstr: "create bookmark",
		},
		{
			name:      "create bookmark missing url",
			wantErr:   true,
			errSubstr: "required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := bookmarkCreateCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, tt.errSubstr, tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkListRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
	}{
		{
			name: "list bookmarks table output",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[{"id":"bm-1","title":"Demo","content":{"url":"https://example.com"}}]}`)))
			},
			wantSubstr: "[bm-1]",
		},
		{
			name: "list bookmarks empty",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[]}`)))
			},
			wantSubstr: "No bookmarks found",
		},
		{
			name: "list bookmarks json output",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[{"id":"bm-2","content":{"url":"https://flowbot.io"}}]}`)))
			},
			args:       []string{"--output", "json"},
			wantSubstr: `"id": "bm-2"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, tt.handler)
			out := runCommand(t, bookmarkListCommand(), tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkGetRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "get bookmark success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/karakeep/bm-9", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"id":"bm-9","title":"Saved","content":{"url":"https://example.com"}}`)))
			},
			args:       []string{"bm-9"},
			wantSubstr: "ID:          bm-9",
		},
		{
			name:    "get bookmark missing id",
			wantErr: true,
		},
		{
			name: "get bookmark json output",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"id":"bm-3","content":{"url":"https://a.test"}}`)))
			},
			args:       []string{"bm-3", "--output", "json"},
			wantSubstr: `"id": "bm-3"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := bookmarkGetCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "required", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkCheckURLRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
	}{
		{
			name: "url already bookmarked",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarkId":"bm-5"}`)))
			},
			args:       []string{"--url", "https://example.com/page"},
			wantSubstr: "URL is bookmarked",
		},
		{
			name: "url not bookmarked",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{}`)))
			},
			args:       []string{"--url", "https://example.com/new"},
			wantSubstr: "URL is not bookmarked",
		},
		{
			name: "check url with query encoding",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "url=")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{}`)))
			},
			args:       []string{"--url", "https://example.com/q?x=1"},
			wantSubstr: "not bookmarked",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, tt.handler)
			out := runCommand(t, bookmarkCheckUrlCommand(), tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkSearchRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
	}{
		{
			name: "search returns results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "q=go")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[{"id":"bm-1","title":"Go tips","content":{"url":"https://go.dev"}}]}`)))
			},
			args:       []string{"--query", "go"},
			wantSubstr: "Found 1 bookmark",
		},
		{
			name: "search empty results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[]}`)))
			},
			args:       []string{"--query", "missing"},
			wantSubstr: "No bookmarks found",
		},
		{
			name: "search json output",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"bookmarks":[{"id":"bm-7","content":{"url":"https://x.test"}}]}`)))
			},
			args:       []string{"--query", "x", "--output", "json"},
			wantSubstr: `"id": "bm-7"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, tt.handler)
			out := runCommand(t, bookmarkSearchCommand(), tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkDeleteRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantSubstr string
	}{
		{
			name: "delete with yes skips prompt",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"archived":true}`)))
			},
			wantSubstr: "Bookmark archived: bm-del",
		},
		{
			name: "delete archives via patch",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"archived":true}`)))
			},
			wantSubstr: "Bookmark archived",
		},
		{
			name: "delete yes flag required path",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"archived":true}`)))
			},
			wantSubstr: "bm-yes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, tt.handler)
			cmd := bookmarkDeleteCommand()
			id := "bm-del"
			if tt.name == "delete yes flag required path" {
				id = "bm-yes"
			}
			out := runCommand(t, cmd, "--yes", id)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestKanbanListRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
	}{
		{
			name: "list active tasks",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "project_id=1")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[{"id":1,"title":"Task A","column_title":"Todo","is_active":1}]`)))
			},
			wantSubstr: "Task A",
		},
		{
			name: "list all tasks",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.NotContains(t, r.URL.RawQuery, "status_id")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[]`)))
			},
			args:       []string{"--status", "all"},
			wantSubstr: "No kanban tasks found",
		},
		{
			name: "list tasks json output",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[{"id":2,"title":"Task B","is_active":1}]`)))
			},
			args:       []string{"--output", "json"},
			wantSubstr: `"title": "Task B"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, tt.handler)
			out := runCommand(t, kanbanListCommand(), tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestKanbanGetRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "get task success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/kanboard/42", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"id":42,"title":"Fix bug","is_active":1,"date_creation":1704067200}`)))
			},
			args:       []string{"42"},
			wantSubstr: "Title:       Fix bug",
		},
		{
			name:    "get task missing id",
			wantErr: true,
		},
		{
			name:    "get task invalid id",
			args:    []string{"abc"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := kanbanGetCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestKanbanCreateRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "create task success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"id":99}`)))
			},
			args:       []string{"--title", "New task"},
			wantSubstr: "Task created: ID=99",
		},
		{
			name:    "create task missing title",
			wantErr: true,
		},
		{
			name: "create task api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"bad request"}`))
			},
			args:    []string{"--title", "X"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := kanbanCreateCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestMemoRunEHandlers(t *testing.T) {
	tests := []struct {
		name       string
		cmd        func() *cobra.Command
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "memo create success",
			cmd:  memoCreateCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":{"name":"memos/1","content":"Hello"}}`)))
			},
			args:       []string{"--content", "Hello"},
			wantSubstr: "Memo created: memos/1",
		},
		{
			name: "memo list table output",
			cmd:  memoListCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":[{"name":"memos/2","content":"Snippet text"}],"page":{"limit":20,"has_more":false}}`)))
			},
			wantSubstr: "[memos/2]",
		},
		{
			name: "memo list empty",
			cmd:  memoListCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":[],"page":{"limit":20,"has_more":false}}`)))
			},
			wantSubstr: "No memos found",
		},
		{
			name: "memo get success",
			cmd:  memoGetCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "name=memos%2F3")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":{"name":"memos/3","content":"Body","visibility":"PRIVATE"}}`)))
			},
			args:       []string{"memos/3"},
			wantSubstr: "Name:       memos/3",
		},
		{
			name: "memo health healthy",
			cmd:  memoHealthCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/memos/health", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":true}`)))
			},
			wantSubstr: "Memo backend is healthy",
		},
		{
			name: "memo delete with yes",
			cmd:  memoDeleteCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
			args:       []string{"--yes", "memos/9"},
			wantSubstr: "Memo deleted: memos/9",
		},
		{
			name:    "memo get missing name",
			cmd:     memoGetCommand,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := tt.cmd()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "required", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestHubRunEHandlers(t *testing.T) {
	tests := []struct {
		name       string
		cmd        func() *cobra.Command
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
		errSubstr  string
	}{
		{
			name: "hub apps list table",
			cmd:  hubAppsListCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/hub/apps", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[{"name":"redis","status":"running","health":"healthy"}]`)))
			},
			wantSubstr: "redis",
		},
		{
			name: "hub apps list empty",
			cmd:  hubAppsListCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[]`)))
			},
			wantSubstr: "No apps registered",
		},
		{
			name: "hub apps status",
			cmd:  hubAppsStatusCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/hub/apps/postgres/status", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"name":"postgres","status":"running"}`)))
			},
			args:       []string{"postgres"},
			wantSubstr: "Status: running",
		},
		{
			name: "hub apps logs",
			cmd:  hubAppsLogsCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "tail=")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"name":"app","logs":["line-1","line-2"]}`)))
			},
			args:       []string{"app", "--tail", "2"},
			wantSubstr: "line-1",
		},
		{
			name: "hub capabilities json",
			cmd:  hubCapabilitiesCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`[{"type":"bookmark","app":"karakeep","healthy":true}]`)))
			},
			args:       []string{"--output", "json"},
			wantSubstr: `"type": "bookmark"`,
		},
		{
			name: "hub health healthy",
			cmd:  hubHealthCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"status":"healthy","timestamp":"2026-01-01T00:00:00Z"}`)))
			},
			wantSubstr: "Hub Status: healthy",
		},
		{
			name: "hub health unhealthy returns error",
			cmd:  hubHealthCommand,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"status":"degraded"}`)))
			},
			wantErr:   true,
			errSubstr: "hub status is degraded",
		},
		{
			name:    "hub apps status missing name",
			cmd:     hubAppsStatusCommand,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := tt.cmd()
			if tt.wantErr {
				runCommandExpectError(t, cmd, tt.errSubstr, tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestConfigRunEHandlers(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) *cobra.Command
		args       []string
		wantSubstr string
		wantErr    bool
		errSubstr  string
	}{
		{
			name: "config get server-url from store",
			setup: func(t *testing.T) *cobra.Command {
				tmpDir := t.TempDir()
				t.Setenv("HOME", tmpDir)
				cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
				require.NoError(t, os.MkdirAll(cfgDir, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://stored.example.com"), 0600))
				return configGetCommand()
			},
			args:       []string{"server-url"},
			wantSubstr: "https://stored.example.com",
		},
		{
			name: "config get debug off",
			setup: func(t *testing.T) *cobra.Command {
				tmpDir := t.TempDir()
				t.Setenv("HOME", tmpDir)
				return configGetCommand()
			},
			args:       []string{"debug"},
			wantSubstr: "off",
		},
		{
			name: "config set server-url",
			setup: func(t *testing.T) *cobra.Command {
				tmpDir := t.TempDir()
				t.Setenv("HOME", tmpDir)
				return configSetCommand()
			},
			args:       []string{"server-url", "https://new.example.com"},
			wantSubstr: "Configuration 'server-url' set",
		},
		{
			name: "config list shows token stored",
			setup: func(t *testing.T) *cobra.Command {
				tmpDir := t.TempDir()
				t.Setenv("HOME", tmpDir)
				cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
				require.NoError(t, os.MkdirAll(cfgDir, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("secret"), 0600))
				require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://s.test"), 0600))
				return configListCommand()
			},
			wantSubstr: "token: [stored]",
		},
		{
			name: "config get unknown key",
			setup: func(_ *testing.T) *cobra.Command {
				return configGetCommand()
			},
			args:      []string{"unknown-key"},
			wantErr:   true,
			errSubstr: "unknown configuration key",
		},
		{
			name: "config set debug on",
			setup: func(t *testing.T) *cobra.Command {
				tmpDir := t.TempDir()
				t.Setenv("HOME", tmpDir)
				return configSetCommand()
			},
			args:       []string{"debug", "on"},
			wantSubstr: "Configuration 'debug' set to 'on'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setup(t)
			wireClientFlags(cmd)
			cmd.SetIn(bytes.NewReader(nil))
			cmd.SetContext(context.Background())
			cmd.SetArgs(tt.args)
			out, err := captureStdout(t, cmd.Execute)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestVersionRunE(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{name: "prints version string", version: "9.9.9-test"},
		{name: "prints semver version", version: "1.2.3"},
		{name: "prints dev version", version: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runCommand(t, VersionCommand(tt.version))
			assert.Contains(t, out, fmt.Sprintf("flowbot version %s", tt.version))
		})
	}
}

func TestKanbanUpdateDeleteMoveRunE(t *testing.T) {
	tests := []struct {
		name       string
		cmd        func() *cobra.Command
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "update task success",
			cmd:  kanbanUpdateCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"success":true}`)))
			},
			args:       []string{"7", "--title", "Updated"},
			wantSubstr: "Task updated: 7",
		},
		{
			name: "delete task with yes",
			cmd:  kanbanDeleteCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"success":true}`)))
			},
			args:       []string{"--yes", "8"},
			wantSubstr: "Task closed: 8",
		},
		{
			name: "move task success",
			cmd:  kanbanMoveCommand,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"success":true}`)))
			},
			args:       []string{"9", "--column", "3"},
			wantSubstr: "Task moved: 9 -> column 3",
		},
		{
			name:    "move task missing column",
			cmd:     kanbanMoveCommand,
			args:    []string{"9"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := tt.cmd()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestBookmarkArchiveRunE(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		getBody    string
		patchBody  string
		wantSubstr string
	}{
		{
			name:       "archive unarchived bookmark",
			id:         "bm-1",
			getBody:    okJSON(`{"id":"bm-1","archived":false,"content":{"url":"https://example.com"}}`),
			patchBody:  okJSON(`{"archived":true}`),
			wantSubstr: "Bookmark archived: bm-1",
		},
		{
			name:       "archive toggles archived bookmark",
			id:         "bm-2",
			getBody:    okJSON(`{"id":"bm-2","archived":true,"content":{"url":"https://example.com"}}`),
			patchBody:  okJSON(`{"archived":false}`),
			wantSubstr: "Bookmark unarchived: bm-2",
		},
		{
			name:       "archive yes skips prompt",
			id:         "bm-3",
			getBody:    okJSON(`{"id":"bm-3","archived":false,"content":{"url":"https://x.test"}}`),
			patchBody:  okJSON(`{"archived":true}`),
			wantSubstr: "Bookmark archived: bm-3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockCLI(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					_, _ = w.Write([]byte(tt.getBody))
				case http.MethodPatch:
					_, _ = w.Write([]byte(tt.patchBody))
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			})
			out := runCommand(t, bookmarkArchiveCommand(), "--yes", tt.id)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestMemoUpdateRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "update memo content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":{"name":"memos/4","content":"Updated","pinned":false}}`)))
			},
			args:       []string{"memos/4", "--content", "Updated"},
			wantSubstr: "Memo updated: memos/4",
		},
		{
			name:    "update memo missing fields",
			args:    []string{"memos/4"},
			wantErr: true,
		},
		{
			name: "update memo pinned",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"data":{"name":"memos/5","content":"X","pinned":true}}`)))
			},
			args:       []string{"memos/5", "--pinned"},
			wantSubstr: "Memo updated [pinned]: memos/5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := memoUpdateCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "at least one", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}

func TestHubRestartRunE(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name: "restart app success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/hub/apps/redis/restart", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(okJSON(`{"status":"restarting"}`)))
			},
			args:       []string{"redis"},
			wantSubstr: "App redis:",
		},
		{
			name:    "restart app missing name",
			wantErr: true,
		},
		{
			name: "restart app api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"restart failed"}`))
			},
			args:    []string{"broken"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				setupMockCLI(t, tt.handler)
			}
			cmd := hubAppsRestartCommand()
			if tt.wantErr {
				runCommandExpectError(t, cmd, "", tt.args...)
				return
			}
			out := runCommand(t, cmd, tt.args...)
			assert.Contains(t, out, tt.wantSubstr)
		})
	}
}
