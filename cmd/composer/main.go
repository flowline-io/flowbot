// Package main is the entry point for the Flowbot composer CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/composer/action/admin"
	"github.com/flowline-io/flowbot/cmd/composer/action/skills"
	"github.com/flowline-io/flowbot/cmd/composer/action/webdoc"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	command := NewCommand()
	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "composer",
		Short:        "tool cli",
		Version:      version.Buildtags,
		SilenceUsage: true,
	}
	rootCmd.SetVersionTemplate("version={{.Version}}\n")

	rootCmd.AddCommand(admin.AdminCommand())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "webdoc",
		Short: "website documentation from markdown sources",
		RunE:  webdoc.WebDocAction,
	})

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "generate SKILL.md files for CLI capabilities",
		RunE:  skills.SkillsAction,
	}
	skillsCmd.Flags().String("output", "./docs/skills", "output directory for SKILL.md files")
	rootCmd.AddCommand(skillsCmd)

	return rootCmd
}
