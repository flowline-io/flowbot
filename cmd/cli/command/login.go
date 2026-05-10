package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flowline-io/flowbot/cmd/cli/store"
)

// LoginCommand returns the login command
func LoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save access token for Flowbot server communication",
		Long:  "Save the access token used to authenticate with the Flowbot server API. The token is sent as X-AccessToken header in all API requests.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			profile, _ := cmd.Flags().GetString("profile")

			serverURL, _ := cmd.Flags().GetString("server-url")
			if serverURL != "" {
				if err := store.SaveServerURL(serverURL, profile); err != nil {
					return fmt.Errorf("save server URL: %w", err)
				}
				_, _ = fmt.Printf("Server URL saved: %s\n", serverURL)
			}

			token, _ := cmd.Flags().GetString("token")
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
	cmd.Flags().StringP("token", "t", "", "Access token for Flowbot server API")
	return cmd
}
