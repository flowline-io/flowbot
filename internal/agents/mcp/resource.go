package mcp

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Resources(s *server.MCPServer) {
	// Static resource example - exposing a README file
	resource := mcp.NewResource(
		"docs://readme",
		"Project README",
		mcp.WithResourceDescription("The project's README file"),
		mcp.WithMIMEType("text/markdown"),
	)

	// Add resource with its handler
	s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		content, err := os.ReadFile("README.md")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "docs://readme",
				MIMEType: "text/markdown",
				Text:     string(content),
			},
		}, nil
	})

	// Dynamic resource example - user profiles by ID
	template := mcp.NewResourceTemplate(
		"users://{id}/profile",
		"User Profile",
		mcp.WithTemplateDescription("Returns user profile information"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	// Add template with its handler
	s.AddResourceTemplate(template, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract ID from the URI using regex matching
		// The server automatically matches URIs to templates
		userID := extractIDFromURI(request.Params.URI)

		profile, err := getUserProfile(userID) // Your DB/API call here
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     profile,
			},
		}, nil
	})
}

func getUserProfile(_ any) (string, error) {
	return "{\"name\": \"John Doe\", \"age\": 30}", nil
}

func extractIDFromURI(_ string) any {
	return "1"
}
