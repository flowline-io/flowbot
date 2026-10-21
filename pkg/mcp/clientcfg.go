package mcp

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
)

var snippetJSON = sonic.Config{SortMapKeys: true}.Froze()

// Client is a supported MCP client snippet target.
type Client string

const (
	// ClientCodex renders a Codex config.toml table.
	ClientCodex Client = "codex"
	// ClientClaude renders a claude mcp add command.
	ClientClaude Client = "claude"
	// ClientCursor renders a Cursor mcp.json fragment.
	ClientCursor Client = "cursor"
	// ClientGeneric renders a transport-neutral JSON fragment.
	ClientGeneric Client = "generic"
)

// SnippetOptions are the inputs for a client config snippet.
type SnippetOptions struct {
	// Name is the MCP server id. Empty uses flowbot.
	Name string
	// URL is the Streamable HTTP endpoint.
	URL string
	// TokenEnv is the environment variable that holds the bearer token. Empty uses FLOWBOT_MCP_TOKEN.
	TokenEnv string
	// Scope is project or user. Empty uses user for Claude and project for Cursor.
	Scope string
	// Approval is the Codex default_tools_approval_mode. Empty uses writes. Only Codex accepts it.
	Approval string
}

// Snippet is one printable client configuration.
type Snippet struct {
	Client  string `json:"client"`
	Target  string `json:"target"`
	Snippet string `json:"snippet"`
	// ServerCard is the public discovery URL for URL. Empty when URL is not http or https.
	ServerCard string `json:"server_card,omitempty"`
}

// RenderSnippet builds a paste-ready client snippet. The token value is never included.
// ServerCard is set when URL is an HTTP endpoint. The snippet text does not include it.
func RenderSnippet(client Client, opt SnippetOptions) (Snippet, error) {
	name, tokenEnv, endpoint, approval, err := normalizeSnippet(opt)
	if err != nil {
		return Snippet{}, err
	}
	var snippet Snippet
	switch client {
	case ClientCodex:
		snippet, err = renderCodex(name, endpoint, tokenEnv, approval, opt.Scope)
	case ClientClaude:
		snippet, err = renderClaude(name, endpoint, tokenEnv, opt)
	case ClientCursor:
		snippet, err = renderCursor(name, endpoint, tokenEnv, opt)
	case ClientGeneric:
		snippet, err = renderGeneric(endpoint, tokenEnv, opt.Approval)
	default:
		return Snippet{}, errors.New("client must be codex, claude, cursor, or generic")
	}
	if err != nil {
		return Snippet{}, err
	}
	snippet.ServerCard = ServerCardURLOrEmpty(endpoint)
	return snippet, nil
}

func normalizeSnippet(opt SnippetOptions) (name, tokenEnv, endpoint, approval string, err error) {
	name = strings.TrimSpace(opt.Name)
	if name == "" {
		name = "flowbot"
	}
	tokenEnv = strings.TrimSpace(opt.TokenEnv)
	if tokenEnv == "" {
		tokenEnv = "FLOWBOT_MCP_TOKEN"
	}
	endpoint = strings.TrimSpace(opt.URL)
	if endpoint == "" {
		return "", "", "", "", errors.New("mcp url is required")
	}
	if _, err = url.ParseRequestURI(endpoint); err != nil {
		return "", "", "", "", errors.New("mcp url is invalid")
	}
	approval = strings.TrimSpace(opt.Approval)
	if approval == "" {
		approval = "writes"
	}
	return name, tokenEnv, endpoint, approval, nil
}

func renderCodex(name, endpoint, tokenEnv, approval, scope string) (Snippet, error) {
	scope = strings.TrimSpace(scope)
	if scope != "" && scope != "user" && scope != "project" {
		return Snippet{}, errors.New("scope must be project or user")
	}
	return Snippet{
		Client:  string(ClientCodex),
		Target:  "~/.codex/config.toml",
		Snippet: codexSnippet(name, endpoint, tokenEnv, approval),
	}, nil
}

func renderClaude(name, endpoint, tokenEnv string, opt SnippetOptions) (Snippet, error) {
	if strings.TrimSpace(opt.Approval) != "" {
		return Snippet{}, errors.New("approval is only valid for codex")
	}
	scope := strings.TrimSpace(opt.Scope)
	if scope == "" {
		scope = "user"
	}
	if scope != "user" && scope != "project" {
		return Snippet{}, errors.New("scope must be project or user")
	}
	return Snippet{
		Client:  string(ClientClaude),
		Target:  "claude mcp add --scope " + scope,
		Snippet: claudeSnippet(name, endpoint, tokenEnv, scope),
	}, nil
}

func renderCursor(name, endpoint, tokenEnv string, opt SnippetOptions) (Snippet, error) {
	if strings.TrimSpace(opt.Approval) != "" {
		return Snippet{}, errors.New("approval is only valid for codex")
	}
	scope := strings.TrimSpace(opt.Scope)
	if scope == "" {
		scope = "project"
	}
	if scope != "user" && scope != "project" {
		return Snippet{}, errors.New("scope must be project or user")
	}
	target := ".cursor/mcp.json"
	if scope == "user" {
		target = "~/.cursor/mcp.json"
	}
	body, err := cursorSnippet(name, endpoint, tokenEnv)
	if err != nil {
		return Snippet{}, err
	}
	return Snippet{Client: string(ClientCursor), Target: target, Snippet: body}, nil
}

func renderGeneric(endpoint, tokenEnv, approval string) (Snippet, error) {
	if strings.TrimSpace(approval) != "" {
		return Snippet{}, errors.New("approval is only valid for codex")
	}
	body, err := genericSnippet(endpoint, tokenEnv)
	if err != nil {
		return Snippet{}, err
	}
	return Snippet{Client: string(ClientGeneric), Target: "generic", Snippet: body}, nil
}

func codexSnippet(name, endpoint, tokenEnv, approval string) string {
	return fmt.Sprintf(`[mcp_servers.%s]
url = "%s"
bearer_token_env_var = "%s"
default_tools_approval_mode = "%s"
`, name, endpoint, tokenEnv, approval)
}

func claudeSnippet(name, endpoint, tokenEnv, scope string) string {
	return fmt.Sprintf("claude mcp add --transport http --scope %s %s %s \\\n  --header \"Authorization: Bearer ${%s}\"\n", scope, name, endpoint, tokenEnv)
}

func cursorSnippet(name, endpoint, tokenEnv string) (string, error) {
	doc := map[string]any{
		"mcpServers": map[string]any{
			name: map[string]any{
				"url": endpoint,
				"headers": map[string]any{
					"Authorization": "Bearer ${env:" + tokenEnv + "}",
				},
			},
		},
	}
	return marshalSnippet(doc)
}

func genericSnippet(endpoint, tokenEnv string) (string, error) {
	doc := map[string]any{
		"transport": "streamable-http",
		"url":       endpoint,
		"headers": map[string]any{
			"Authorization": "Bearer <" + tokenEnv + ">",
		},
	}
	return marshalSnippet(doc)
}

func marshalSnippet(doc any) (string, error) {
	raw, err := snippetJSON.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
