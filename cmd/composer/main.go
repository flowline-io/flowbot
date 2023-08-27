package main

import (
	"fmt"
	"github.com/sysatom/flowbot/cmd/composer/action/dao"
	"github.com/sysatom/flowbot/cmd/composer/action/generator"
	"github.com/sysatom/flowbot/cmd/composer/action/migrate"
	"github.com/sysatom/flowbot/pkg/version"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	command := NewCommand()
	if err := command.Run(os.Args); err != nil {
		panic(err)
	}
}

func NewCommand() *cli.App {
	cli.VersionPrinter = func(_ *cli.Context) {
		fmt.Printf("version=%s\n", version.CurrentVersion)
	}
	return &cli.App{
		Name:                 "composer",
		Usage:                "chatbot tool cli",
		EnableBashCompletion: true,
		Version:              version.CurrentVersion,
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "migrate tool",
				Subcommands: []*cli.Command{
					{
						Name:  "import",
						Usage: "import database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "config",
								Value: "./flowbot.conf",
								Usage: "config of the database connection",
							},
						},
						Action: migrate.ImportAction,
					},
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
						Value: "./flowbot.conf",
						Usage: "config of the database connection",
					},
				},
				Action: dao.GenerationAction,
			},
		},
	}
}
