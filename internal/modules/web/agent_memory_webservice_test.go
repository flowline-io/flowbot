package web

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
)

func withMemoryWorkspace(t *testing.T, fn func()) {
	t.Helper()
	orig := pkgconfig.App.ChatAgent
	root := t.TempDir()
	pkgconfig.App.ChatAgent = pkgconfig.ChatAgentConfig{
		ChatModel: "gpt-test",
		Workspace: root,
	}
	t.Cleanup(func() { pkgconfig.App.ChatAgent = orig })
	fn()
}

func TestAgentMemoryWebserviceCRUD(t *testing.T) {
	withMemoryWorkspace(t, func() {
		app, _ := setupTestApp()
		defer func() { handler = moduleHandler{}; config = configType{} }()

		tests := []struct {
			name       string
			method     string
			path       string
			body       string
			wantStatus int
		}{
			{name: "list files", method: http.MethodGet, path: "/service/web/agent-memory/files?scope=my-pipeline", wantStatus: http.StatusOK},
			{name: "read empty", method: http.MethodGet, path: "/service/web/agent-memory/content?scope=my-pipeline&file=MEMORIES.md", wantStatus: http.StatusOK},
			{name: "write content", method: http.MethodPut, path: "/service/web/agent-memory/content", body: `{"scope":"my-pipeline","file":"MEMORIES.md","content":"hello"}`, wantStatus: http.StatusOK},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var body io.Reader = http.NoBody
				if tt.body != "" {
					body = bytes.NewBufferString(tt.body)
				}
				req := httptest.NewRequest(tt.method, tt.path, body)
				req.Header.Set("Cookie", "accessToken=test-token")
				AttachCSRFForTest(req)
				if tt.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				resp, err := app.Test(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, tt.wantStatus, resp.StatusCode)
			})
		}
	})
}

func TestAgentMemoryWebserviceUnauthenticated(t *testing.T) {
	withMemoryWorkspace(t, func() {
		app, _ := setupTestApp()
		defer func() { handler = moduleHandler{}; config = configType{} }()

		req := httptest.NewRequest(http.MethodGet, "/service/web/agent-memory/files?scope=test", http.NoBody)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	})
}

func TestAgentMemoryWebserviceRequiresWorkspace(t *testing.T) {
	orig := pkgconfig.App.ChatAgent
	pkgconfig.App.ChatAgent.Workspace = ""
	t.Cleanup(func() { pkgconfig.App.ChatAgent = orig })

	app, _ := setupTestApp()
	defer func() { handler = moduleHandler{}; config = configType{} }()

	req := httptest.NewRequest(http.MethodGet, "/service/web/agent-memory/files?scope=test", http.NoBody)
	req.Header.Set("Cookie", "accessToken=test-token")
	AttachCSRFForTest(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
