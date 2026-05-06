package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/composer/action/dao"
	"github.com/flowline-io/flowbot/cmd/composer/action/doc"
	"github.com/flowline-io/flowbot/cmd/composer/action/skills"
	"github.com/flowline-io/flowbot/cmd/composer/action/webdoc"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
	"github.com/urfave/cli/v3"
)

func main() {
	command := NewCommand()
	if err := command.Run(context.Background(), os.Args); err != nil {
		flog.Panic("%s", err.Error())
	}
}

func NewCommand() *cli.Command {
	cli.VersionPrinter = func(_ *cli.Command) {
		_, _ = fmt.Printf("version=%s\n", version.Buildtags)
	}
	return &cli.Command{
		Name:                  "composer",
		Usage:                 "tool cli",
		EnableShellCompletion: true,
		Version:               version.Buildtags,
		Commands: []*cli.Command{
			{
				Name:  "dao",
				Usage: "dao generator",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "./flowbot.yaml",
						Usage: "config of the database connection",
					},
				},
				Action: dao.GenerationAction,
			},
			{
				Name:   "webdoc",
				Usage:  "website documentation from markdown sources",
				Action: webdoc.WebDocAction,
			},
			{
				Name:   "skills",
				Usage:  "generate SKILL.md files for CLI capabilities",
				Action: skills.SkillsAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "output",
						Value: "./docs/skills",
						Usage: "output directory for SKILL.md files",
					},
				},
			},
			{
				Name:  "doc",
				Usage: "database schema documentation",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "./flowbot.yaml",
						Usage: "config of the database connection",
					},
					&cli.StringFlag{
						Name:  "database",
						Value: "flowbot",
						Usage: "database name",
					},
				},
				Action: doc.SchemaAction,
			},
		},
	}
}
