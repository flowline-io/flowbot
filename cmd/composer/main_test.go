package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func subcommandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	return names
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()

	require.Equal(t, "composer", cmd.Use)
	require.Equal(t, "tool cli", cmd.Short)
	require.NotEmpty(t, cmd.Version)
	require.True(t, cmd.HasSubCommands())

	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "admin")
	require.Contains(t, subNames, "dao")
	require.Contains(t, subNames, "webdoc")
	require.Contains(t, subNames, "skills")
	require.Contains(t, subNames, "doc")
}

func TestDaoCommand(t *testing.T) {
	cmd := NewCommand()
	daoCmd := findSubcommand(cmd, "dao")
	require.NotNil(t, daoCmd)
	require.Equal(t, "dao", daoCmd.Use)
	require.NotNil(t, daoCmd.RunE)

	configFlag := daoCmd.Flags().Lookup("config")
	require.NotNil(t, configFlag)
	val, _ := daoCmd.Flags().GetString("config")
	require.Equal(t, "./flowbot.yaml", val)
}

func TestWebdocCommand(t *testing.T) {
	cmd := NewCommand()
	webdocCmd := findSubcommand(cmd, "webdoc")
	require.NotNil(t, webdocCmd)
	require.Equal(t, "webdoc", webdocCmd.Use)
	require.NotNil(t, webdocCmd.RunE)
}

func TestSkillsCommand(t *testing.T) {
	cmd := NewCommand()
	skillsCmd := findSubcommand(cmd, "skills")
	require.NotNil(t, skillsCmd)
	require.Equal(t, "skills", skillsCmd.Use)
	require.NotNil(t, skillsCmd.RunE)

	outputFlag := skillsCmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	val, _ := skillsCmd.Flags().GetString("output")
	require.Equal(t, "./docs/skills", val)
}

func TestDocCommand(t *testing.T) {
	cmd := NewCommand()
	docCmd := findSubcommand(cmd, "doc")
	require.NotNil(t, docCmd)
	require.Equal(t, "doc", docCmd.Use)
	require.NotNil(t, docCmd.RunE)

	configFlag := docCmd.Flags().Lookup("config")
	require.NotNil(t, configFlag)
	val, _ := docCmd.Flags().GetString("config")
	require.Equal(t, "./flowbot.yaml", val)

	dbFlag := docCmd.Flags().Lookup("database")
	require.NotNil(t, dbFlag)
	dbVal, _ := docCmd.Flags().GetString("database")
	require.Equal(t, "flowbot", dbVal)
}

func TestAdminCommand(t *testing.T) {
	cmd := NewCommand()
	adminCmd := findSubcommand(cmd, "admin")
	require.NotNil(t, adminCmd)
	require.True(t, adminCmd.HasSubCommands())

	subNames := subcommandNames(adminCmd)
	require.Contains(t, subNames, "token")
}

func TestAdminTokenCreateCommand(t *testing.T) {
	cmd := NewCommand()
	adminCmd := findSubcommand(cmd, "admin")
	tokenCmd := findSubcommand(adminCmd, "token")
	require.NotNil(t, tokenCmd)
	require.True(t, tokenCmd.HasSubCommands())

	createCmd := findSubcommand(tokenCmd, "create")
	require.NotNil(t, createCmd)
	require.NotNil(t, createCmd.RunE)

	id := createCmd.Flags().Lookup("id")
	require.NotNil(t, id)
	ann := id.Annotations[cobra.BashCompOneRequiredFlag]
	require.NotNil(t, ann)
	require.Contains(t, ann, "true")

	expires := createCmd.Flags().Lookup("expires")
	require.NotNil(t, expires)
	val, _ := createCmd.Flags().GetString("expires")
	require.Equal(t, "0d", val)

	configFlag := createCmd.Flags().Lookup("config")
	require.NotNil(t, configFlag)
	val, _ = createCmd.Flags().GetString("config")
	require.Equal(t, "./flowbot.yaml", val)
}
