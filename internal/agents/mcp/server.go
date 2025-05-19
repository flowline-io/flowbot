package mcp

import (
	"net/http"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/version"
	"github.com/mark3labs/mcp-go/server"
)

func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Flowbot MCP Server",
		version.Buildtags,
	)

	// register resources
	Resources(s)
	// register tools
	Tools(s)
	// register prompts
	Prompts(s)

	return s
}

func NewSSEServer(_ config.Type, mcpServer *server.MCPServer) *server.SSEServer {
	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
			return "/mcp/"
		}),
		server.WithBaseURL(config.App.Flowbot.URL),
		server.WithUseFullURLForMessageEndpoint(true),
	)

	return sseServer
}
