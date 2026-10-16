package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/types"
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
			}

			token, _ := cmd.Flags().GetString("token")
			if token == "" {
				if utils.NoInput(cmd) {
					return types.Errorf(types.ErrInvalidArgument, "token is required")
				}
				if _, err := fmt.Fprint(cmd.OutOrStdout(), "Enter your API token: "); err != nil {
					return fmt.Errorf("write token prompt: %w", err)
				}
				byteToken, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("read token: %w", err)
				}
				token = string(byteToken)
				if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
					return fmt.Errorf("write token prompt: %w", err)
				}
			}

			if token == "" {
				return types.Errorf(types.ErrInvalidArgument, "token is required")
			}

			if err := store.SaveToken(token, profile); err != nil {
				return err
			}
			if IsJSON(cmd) {
				data := map[string]any{}
				if serverURL != "" {
					data["server_url"] = serverURL
				}
				return render(cmd, data)
			}
			if serverURL != "" {
				if err := writef(cmd, "Server URL saved: %s\n", serverURL); err != nil {
					return err
				}
			}
			return writef(cmd, "Login successful. Token saved.\n")
		},
	}
	cmd.Flags().StringP("token", "t", "", "Access token for Flowbot server API")
	return cmd
}
