package main

import (
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/composer/action/admin"
	"github.com/flowline-io/flowbot/cmd/composer/action/dao"
	"github.com/flowline-io/flowbot/cmd/composer/action/doc"
	"github.com/flowline-io/flowbot/cmd/composer/action/skills"
	"github.com/flowline-io/flowbot/cmd/composer/action/webdoc"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	command := NewCommand()
	if err := command.Execute(); err != nil {
		flog.Panic("%s", err.Error())
	}
}

func NewCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "composer",
		Short:   "tool cli",
		Version: version.Buildtags,
	}
	rootCmd.SetVersionTemplate("version={{.Version}}\n")

	rootCmd.AddCommand(admin.AdminCommand())

	daoCmd := &cobra.Command{
		Use:   "dao",
		Short: "dao generator",
		RunE:  dao.GenerationAction,
	}
	daoCmd.Flags().String("config", "./flowbot.yaml", "config of the database connection")
	rootCmd.AddCommand(daoCmd)

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

	docCmd := &cobra.Command{
		Use:   "doc",
		Short: "database schema documentation",
		RunE:  doc.SchemaAction,
	}
	docCmd.Flags().String("config", "./flowbot.yaml", "config of the database connection")
	docCmd.Flags().String("database", "flowbot", "database name")
	rootCmd.AddCommand(docCmd)

	return rootCmd
}
