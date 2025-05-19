package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Prompts(s *server.MCPServer) {
	// Simple greeting prompt
	s.AddPrompt(mcp.NewPrompt("greeting",
		mcp.WithPromptDescription("A friendly greeting prompt"),
		mcp.WithArgument("name",
			mcp.ArgumentDescription("Name of the person to greet"),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		name := request.Params.Arguments["name"]
		if name == "" {
			name = "friend"
		}

		return mcp.NewGetPromptResult(
			"A friendly greeting",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleAssistant,
					mcp.NewTextContent(fmt.Sprintf("Hello, %s! How can I help you today?", name)),
				),
			},
		), nil
	})

	// Code review prompt with embedded resource
	s.AddPrompt(mcp.NewPrompt("code_review",
		mcp.WithPromptDescription("Code review assistance"),
		mcp.WithArgument("pr_number",
			mcp.ArgumentDescription("Pull request number to review"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		prNumber := request.Params.Arguments["pr_number"]
		if prNumber == "" {
			return nil, fmt.Errorf("pr_number is required")
		}

		return mcp.NewGetPromptResult(
			"Code review assistance",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent("Review the changes and provide constructive feedback."),
				),
				mcp.NewPromptMessage(
					mcp.RoleAssistant,
					mcp.NewEmbeddedResource(mcp.TextResourceContents{
						URI:      fmt.Sprintf("git://pulls/%s/diff", prNumber),
						MIMEType: "text/x-diff",
					}),
				),
			},
		), nil
	})

	// Database query builder prompt
	s.AddPrompt(mcp.NewPrompt("query_builder",
		mcp.WithPromptDescription("SQL query builder assistance"),
		mcp.WithArgument("table",
			mcp.ArgumentDescription("Name of the table to query"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		tableName := request.Params.Arguments["table"]
		if tableName == "" {
			return nil, fmt.Errorf("table name is required")
		}

		return mcp.NewGetPromptResult(
			"SQL query builder assistance",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent("Help construct efficient and safe queries for the provided schema."),
				),
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewEmbeddedResource(mcp.TextResourceContents{
						URI:      fmt.Sprintf("db://schema/%s", tableName),
						MIMEType: "application/json",
					}),
				),
			},
		), nil
	})
}
