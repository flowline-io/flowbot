package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// LoginCommand returns the login command
func LoginCommand() *cli.Command {
	return &cli.Command{
		Name:        "login",
		Usage:       "Authenticate flowbot with Flowbot server",
		Description: "Authenticate and save credentials locally",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "API token for authentication",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "server-url",
				Usage:    "Flowbot server URL",
				Required: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")

			token := cmd.String("token")
			if token == "" {
				_, _ = fmt.Print("Enter your API token: ")
				byteToken, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("read token: %w", err)
				}
				token = string(byteToken)
				_, _ = fmt.Println()
			}

			if token == "" {
				return fmt.Errorf("token is required")
			}

			if err := store.SaveToken(token, profile); err != nil {
				return err
			}
			_, _ = fmt.Println("Login successful. Token saved.")
			return nil
		},
	}
}
