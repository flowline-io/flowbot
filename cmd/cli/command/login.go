package command

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// LoginCommand returns the login command
func LoginCommand() *cli.Command {
	return &cli.Command{
		Name:        "login",
		Usage:       "Save access token for Flowbot server communication",
		Description: "Save the access token used to authenticate with the Flowbot server API. The token is sent as X-AccessToken header in all API requests.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "Access token for Flowbot server API",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "server-url",
				Usage:    "Flowbot server URL",
				Required: false,
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")

			serverURL := cmd.String("server-url")
			if serverURL != "" {
				if err := store.SaveServerURL(serverURL, profile); err != nil {
					return fmt.Errorf("save server URL: %w", err)
				}
				_, _ = fmt.Printf("Server URL saved: %s\n", serverURL)
			}

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
