package command

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestBookmarkCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "bookmark command has correct use and subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BookmarkCommand()

			require.Equal(t, "bookmark", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "archive")
			require.Contains(t, subNames, "delete")
			require.Contains(t, subNames, "check-url")
			require.Contains(t, subNames, "search")
		})
	}
}

func TestBookmarkCreateRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "bookmark create has required --url flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BookmarkCommand()
			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)

			url := createCmd.Flags().Lookup("url")
			require.NotNil(t, url)
			require.Equal(t, "url", url.Name)
			require.Equal(t, "u", url.Shorthand)

			ann := url.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
			require.Contains(t, ann, "true")
		})
	}
}

func TestBookmarkListCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "bookmark list has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BookmarkCommand()
			listCmd := findSubcommand(cmd, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)

			limit := listCmd.Flags().Lookup("limit")
			require.NotNil(t, limit)
			require.Equal(t, "n", limit.Shorthand)
		})
	}
}

func TestBookmarkSearchCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "bookmark search has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BookmarkCommand()
			searchCmd := findSubcommand(cmd, "search")
			require.NotNil(t, searchCmd)
			require.NotNil(t, searchCmd.RunE)

			query := searchCmd.Flags().Lookup("query")
			require.NotNil(t, query)
			ann := query.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
			require.Contains(t, ann, "true")

			sortOrder := searchCmd.Flags().Lookup("sort-order")
			require.NotNil(t, sortOrder)
			val, _ := searchCmd.Flags().GetString("sort-order")
			require.Equal(t, "relevance", val)

			includeContent := searchCmd.Flags().Lookup("include-content")
			require.NotNil(t, includeContent)
		})
	}
}
