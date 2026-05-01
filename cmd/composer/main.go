package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/composer/action/dao"
	"github.com/flowline-io/flowbot/cmd/composer/action/doc"
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
		Usage:                 "chatbot tool cli",
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
