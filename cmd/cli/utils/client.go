// Package utils provides shared CLI utility functions.
package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/pkg/client"
)

// ResolveServerURL returns the Flowbot server URL from flags, env, or stored config.
// Resolution order: --server-url, FLOWBOT_SERVER_URL, then stored server_url.
func ResolveServerURL(cmd *cobra.Command) (string, error) {
	profile, _ := cmd.Flags().GetString("profile")

	serverURL, _ := cmd.Flags().GetString("server-url")
	if serverURL == "" {
		serverURL = os.Getenv("FLOWBOT_SERVER_URL")
	}
	if serverURL == "" {
		stored, err := store.LoadServerURL(profile)
		if err != nil {
			return "", fmt.Errorf("load server URL: %w", err)
		}
		if stored == "" {
			return "", errors.New("server URL is required (use --server-url, FLOWBOT_SERVER_URL, or 'flowbot config set server-url <url>')")
		}
		serverURL = stored
	}
	return serverURL, nil
}

// NewClient builds an API client from CLI flags, environment variables, and stored config.
// Server URL resolution order: --server-url, FLOWBOT_SERVER_URL, then stored server_url.
// Token resolution order: FLOWBOT_TOKEN, then stored token file.
func NewClient(cmd *cobra.Command) (*client.Client, error) {
	profile, _ := cmd.Flags().GetString("profile")

	serverURL, err := ResolveServerURL(cmd)
	if err != nil {
		return nil, err
	}

	token := os.Getenv("FLOWBOT_TOKEN")
	if token == "" {
		var loadErr error
		token, loadErr = store.LoadToken(profile)
		if loadErr != nil {
			return nil, fmt.Errorf("load token: %w", loadErr)
		}
	}
	if token == "" {
		return nil, errors.New("not logged in (use 'flowbot login' first, or set FLOWBOT_TOKEN)")
	}

	cl := client.NewClient(serverURL, token)
	applyDebug(cmd, profile, cl)
	if err := applyIdempotencyKey(cmd, cl); err != nil {
		return nil, err
	}
	return cl, nil
}

// NewPublicClient builds an API client that can call public endpoints without a token.
// Server URL resolution matches NewClient. Returns an error when no server URL is configured.
func NewPublicClient(cmd *cobra.Command) (*client.Client, error) {
	profile, _ := cmd.Flags().GetString("profile")
	serverURL, err := ResolveServerURL(cmd)
	if err != nil {
		return nil, err
	}
	cl := client.NewClient(serverURL, "")
	applyDebug(cmd, profile, cl)
	return cl, nil
}

func applyDebug(cmd *cobra.Command, profile string, cl *client.Client) {
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
}

const idempotencyReplayedAnnotation = "idempotency_replayed"

func applyIdempotencyKey(cmd *cobra.Command, cl *client.Client) error {
	if cmd == nil || cl == nil {
		return nil
	}
	if cmd.Flags().Lookup("idempotency-key") == nil {
		return nil
	}
	key, err := cmd.Flags().GetString("idempotency-key")
	if err != nil {
		return fmt.Errorf("read idempotency-key: %w", err)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	cl.SetIdempotencyKey(key, func() {
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations[idempotencyReplayedAnnotation] = "true"
	})
	return nil
}
