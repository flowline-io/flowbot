package main

import (
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/composer/action/dao"
	"github.com/flowline-io/flowbot/cmd/composer/action/doc"
	"github.com/flowline-io/flowbot/cmd/composer/action/generator"
	"github.com/flowline-io/flowbot/cmd/composer/action/migrate"
	"github.com/flowline-io/flowbot/version"
	"github.com/urfave/cli/v2"
)

func main() {
	command := NewCommand()
	if err := command.Run(os.Args); err != nil {
		panic(err)
	}
}

func NewCommand() *cli.App {
	cli.VersionPrinter = func(_ *cli.Context) {
		_, _ = fmt.Printf("version=%s\n", version.Buildtags)
	}
	return &cli.App{
		Name:                 "composer",
		Usage:                "chatbot tool cli",
		EnableBashCompletion: true,
		Version:              version.Buildtags,
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "migrate tool",
				Subcommands: []*cli.Command{
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
				Subcommands: []*cli.Command{
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
								Value: cli.NewStringSlice("command"),
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
		},
	}
}
