package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/cli/cmd"
	"github.com/urfave/cli/v3"
)

const (
	appName  = "flowbot"
	appUsage = "Work seamlessly with Flowbot from the command line"
	version  = "1.0.0"
)

func main() {
	rootCmd := &cli.Command{
		Name:    appName,
		Usage:   appUsage,
		Version: version,
		Commands: []*cli.Command{
			cmd.LoginCommand(),
			cmd.BookmarkCommand(),
			cmd.KanbanCommand(),
			cmd.ConfigCommand(),
			cmd.VersionCommand(version),
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
		},
	}

	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
