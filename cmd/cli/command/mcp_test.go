package command

import (
	"bytes"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

func TestMCPConfigSnippets(t *testing.T) {
	t.Parallel()
	opt := pkgmcp.SnippetOptions{URL: "https://flowbot.lan/mcp"}
	cases := []struct {
		client pkgmcp.Client
		want   string
	}{
		{
			client: pkgmcp.ClientCodex,
			want: `[mcp_servers.flowbot]
url = "https://flowbot.lan/mcp"
bearer_token_env_var = "FLOWBOT_MCP_TOKEN"
default_tools_approval_mode = "writes"
`,
		},
		{
			client: pkgmcp.ClientClaude,
			want:   "claude mcp add --transport http --scope user flowbot https://flowbot.lan/mcp \\\n  --header \"Authorization: Bearer ${FLOWBOT_MCP_TOKEN}\"\n",
		},
		{
			client: pkgmcp.ClientCursor,
			want:   `{"mcpServers":{"flowbot":{"headers":{"Authorization":"Bearer ${env:FLOWBOT_MCP_TOKEN}"},"url":"https://flowbot.lan/mcp"}}}`,
		},
		{
			client: pkgmcp.ClientGeneric,
			want:   `{"headers":{"Authorization":"Bearer <FLOWBOT_MCP_TOKEN>"},"transport":"streamable-http","url":"https://flowbot.lan/mcp"}`,
		},
	}
	for _, tt := range cases {
		t.Run(string(tt.client), func(t *testing.T) {
			t.Parallel()
			got, err := pkgmcp.RenderSnippet(tt.client, opt)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Snippet)
			assert.NotContains(t, got.Snippet, "secret-token")
		})
	}
	codex, err := pkgmcp.RenderSnippet(pkgmcp.ClientCodex, opt)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, toml.Unmarshal([]byte(codex.Snippet), &parsed))
	servers, ok := parsed["mcp_servers"].(map[string]any)
	require.True(t, ok)
	flowbot, ok := servers["flowbot"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://flowbot.lan/mcp", flowbot["url"])
	assert.Equal(t, "FLOWBOT_MCP_TOKEN", flowbot["bearer_token_env_var"])
	assert.Equal(t, "writes", flowbot["default_tools_approval_mode"])
}

func TestMCPConfigCommandPrintsCodex(t *testing.T) {
	t.Parallel()
	cmd := MCPCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "--client", "codex", "--url", "https://flowbot.lan/mcp"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), `bearer_token_env_var = "FLOWBOT_MCP_TOKEN"`)
	assert.NotContains(t, buf.String(), "eyJ")
}

func TestMCPConfigRejectsExplicitApprovalOnClaude(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "flowbot"}
	root.AddCommand(MCPCommand())
	root.SetArgs([]string{"mcp", "config", "--client", "claude", "--url", "https://flowbot.lan/mcp", "--approval", "writes"})
	root.SetOut(&bytes.Buffer{})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approval")
}

func TestMCPConfigRejectsApprovalOnCursor(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "flowbot"}
	root.AddCommand(MCPCommand())
	root.SetArgs([]string{"mcp", "config", "--client", "cursor", "--url", "https://flowbot.lan/mcp", "--approval", "writes"})
	root.SetOut(&bytes.Buffer{})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approval")
}
