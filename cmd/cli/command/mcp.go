package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

// MCPCommand returns the mcp parent command.
func MCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Print MCP client configuration",
	}
	cmd.AddCommand(mcpConfigCommand())
	return cmd
}

func mcpConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print a paste-ready MCP client snippet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientName, _ := cmd.Flags().GetString("client")
			name, _ := cmd.Flags().GetString("name")
			tokenEnv, _ := cmd.Flags().GetString("token-env")
			scope, _ := cmd.Flags().GetString("scope")
			approval, _ := cmd.Flags().GetString("approval")
			endpoint, err := mcpConfigURL(cmd)
			if err != nil {
				return err
			}
			opt := pkgmcp.SnippetOptions{
				Name:     name,
				URL:      endpoint,
				TokenEnv: tokenEnv,
				Scope:    scope,
			}
			if cmd.Flags().Changed("approval") {
				opt.Approval = approval
			}
			snippet, err := pkgmcp.RenderSnippet(pkgmcp.Client(clientName), opt)
			if err != nil {
				return err
			}
			if IsJSON(cmd) {
				return PrintJSON(snippet)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), snippet.Snippet)
			return err
		},
	}
	cmd.Flags().String("client", "", "Client: codex, claude, cursor, or generic")
	cmd.Flags().String("name", "flowbot", "MCP server name")
	cmd.Flags().String("url", "", "MCP endpoint (default: configured server URL + /mcp)")
	cmd.Flags().String("token-env", "FLOWBOT_MCP_TOKEN", "Environment variable that holds the bearer token")
	cmd.Flags().String("scope", "", "project or user (claude default user, cursor default project)")
	cmd.Flags().String("approval", "writes", "Codex default_tools_approval_mode")
	cmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	_ = cmd.MarkFlagRequired("client")
	return cmd
}

func mcpConfigURL(cmd *cobra.Command) (string, error) {
	endpoint, _ := cmd.Flags().GetString("url")
	if strings.TrimSpace(endpoint) != "" {
		return strings.TrimSpace(endpoint), nil
	}
	base, err := utils.ResolveServerURL(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(strings.TrimSpace(base), "/") + "/mcp", nil
}
