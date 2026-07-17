// Package utils provides shared CLI utility functions.
package utils

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/pkg/client"
)

// NewClient builds an API client from CLI flags, environment variables, and stored config.
// Server URL resolution order: --server-url, FLOWBOT_SERVER_URL, then stored server_url.
// Token resolution order: FLOWBOT_TOKEN, then stored token file.
func NewClient(cmd *cobra.Command) (*client.Client, error) {
	profile, _ := cmd.Flags().GetString("profile")

	serverURL, _ := cmd.Flags().GetString("server-url")
	if serverURL == "" {
		serverURL = os.Getenv("FLOWBOT_SERVER_URL")
	}
	if serverURL == "" {
		stored, err := store.LoadServerURL(profile)
		if err != nil {
			return nil, fmt.Errorf("load server URL: %w", err)
		}
		if stored == "" {
			return nil, fmt.Errorf("server URL is required (use --server-url, FLOWBOT_SERVER_URL, or 'flowbot config set server-url <url>')")
		}
		serverURL = stored
	}

	token := os.Getenv("FLOWBOT_TOKEN")
	if token == "" {
		var err error
		token, err = store.LoadToken(profile)
		if err != nil {
			return nil, fmt.Errorf("load token: %w", err)
		}
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in (use 'flowbot login' first, or set FLOWBOT_TOKEN)")
	}

	cl := client.NewClient(serverURL, token)

	debug, _ := cmd.Flags().GetBool("debug")
	if !debug {
		debugEnv := os.Getenv("FLOWBOT_DEBUG")
		if debugEnv == "true" || debugEnv == "1" {
			debug = true
		}
	}
	if !debug {
		stored, _ := store.LoadDebug(profile)
		debug = stored
	}
	if debug {
		cl.SetDebug(true)
	}

	return cl, nil
}
