package utils

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/urfave/cli/v3"
)

func NewClient(cmd *cli.Command) (*client.Client, error) {
	profile := cmd.String("profile")

	serverURL := cmd.String("server-url")
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

	token, err := store.LoadToken(profile)
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in (use 'flowbot login' first)")
	}

	cl := client.NewClient(serverURL, token)

	debug := cmd.Bool("debug")
	if !debug {
		stored, _ := store.LoadDebug(profile)
		debug = stored
	}
	if debug {
		cl.SetDebug(true)
	}

	return cl, nil
}
