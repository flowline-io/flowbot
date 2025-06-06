package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/composer/action/dao"
	"github.com/flowline-io/flowbot/cmd/composer/action/doc"
	"github.com/flowline-io/flowbot/cmd/composer/action/generator"
	"github.com/flowline-io/flowbot/cmd/composer/action/migrate"
	"github.com/flowline-io/flowbot/cmd/composer/action/workflow"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
	"github.com/urfave/cli/v3"
)

func main() {
	command := NewCommand()
	if err := command.Run(context.Background(), os.Args); err != nil {
		flog.Panic(err.Error())
	}
}

func NewCommand() *cli.Command {
	cli.VersionPrinter = func(_ *cli.Command) {
		_, _ = fmt.Printf("version=%s\n", version.Buildtags)
	}
	return &cli.Command{
		Name:                  "composer",
		Usage:                 "chatbot tool cli",
		EnableShellCompletion: true,
		Version:               version.Buildtags,
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "migrate tool",
				Commands: []*cli.Command{
					{
						Name:  "migration",
						Usage: "generate migration files",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "name",
								Value: "",
								Usage: "migration name",
							},
						},
						Action: migrate.MigrationAction,
					},
				},
			},
			{
				Name:  "generator",
				Usage: "code generator",
				Commands: []*cli.Command{
					{
						Name:  "bot",
						Usage: "generate bot code files",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "name",
								Value: "",
								Usage: "bot package name",
							},
							&cli.StringSliceFlag{
								Name:  "rule",
								Value: []string{"command"},
								Usage: "rule type",
							},
						},
						Action: generator.BotAction,
					},
					{
						Name:  "vendor",
						Usage: "generate vendor api files",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "name",
								Value: "",
								Usage: "vendor name",
							},
						},
						Action: generator.VendorAction,
					},
				},
			},
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
			{
				Name:  "workflow",
				Usage: "workflow",
				Commands: []*cli.Command{
					{
						Name:  "import",
						Usage: "import workflow yaml",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "config",
								Value: "./flowbot.yaml",
								Usage: "config of the api",
							},
							&cli.StringFlag{
								Name:  "token",
								Value: "",
								Usage: "api access token",
							},
							&cli.StringFlag{
								Name:  "path",
								Value: "",
								Usage: "yaml path",
							},
						},
						Action: workflow.ImportAction,
					},
				},
			},
		},
	}
}
