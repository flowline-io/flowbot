package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
)

func TestMCPSettingsPage(t *testing.T) {
	app, _ := setupTestApp(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
	prev := pkgconfig.App
	pkgconfig.App.MCP.Enabled = false
	t.Cleanup(func() { pkgconfig.App = prev })

	unauth := httptest.NewRequest(http.MethodGet, "/service/web/settings/mcp", http.NoBody)
	resp, err := app.Test(unauth)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status %d", resp.StatusCode)
	}

	req := httptest.NewRequest(http.MethodGet, "/service/web/settings/mcp", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	AttachCSRFForTest(req)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	html := string(body)
	for _, want := range []string{
		`data-testid="mcp-settings-page"`,
		`data-testid="mcp-restart-note"`,
		`class="flowbot-surface mcp-settings"`,
		`class="select select-bordered select-sm w-full"`,
		`data-clip-copy`,
		`data-testid="mcp-snippet-codex"`,
		`data-testid="mcp-snippet-claude"`,
		`data-testid="mcp-snippet-cursor"`,
		`data-testid="mcp-snippet-generic"`,
		"FLOWBOT_MCP_TOKEN",
		`name="csrf_token"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
