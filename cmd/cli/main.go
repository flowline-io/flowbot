package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/flowline-io/flowbot/version"
	"github.com/urfave/cli/v3"
)

const (
	appName  = "flowbot"
	appUsage = "Work seamlessly with Flowbot from the command line"
)

func main() {
	rootCmd := &cli.Command{
		Name:    appName,
		Usage:   appUsage,
		Version: version.Buildtags,
		Commands: []*cli.Command{
			command.LoginCommand(),
			command.HubCommand(),
			command.BookmarkCommand(),
			command.KanbanCommand(),
			command.ReaderCommand(),
			command.ConfigCommand(),
			command.VersionCommand(version.Buildtags),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "profile",
				Usage: "Configuration profile name (e.g. dev)",
			},
			&cli.StringFlag{
				Name:    "server-url",
				Usage:   "Flowbot server URL",
				Sources: cli.EnvVars("FLOWBOT_SERVER_URL"),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug mode (prints HTTP request/response logs)",
				Sources: cli.EnvVars("FLOWBOT_DEBUG"),
			},
		},
	}

	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
