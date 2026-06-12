// Package main is the entry point for the flowbot-chat terminal client.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/chat/app"
	"github.com/flowline-io/flowbot/cmd/chat/utils"
)

func main() {
	root := &cobra.Command{
		Use:          "flowbot-chat",
		Short:        "Chat with the Flowbot Chat Agent in your terminal",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cl, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			profile, _ := cmd.Flags().GetString("profile")
			model := app.NewModel(cl, profile)
			p := app.NewProgram(model)
			if _, err := p.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	root.PersistentFlags().String("profile", "", "Configuration profile name (e.g. dev)")
	root.PersistentFlags().String("server-url", "", "Flowbot server URL")
	root.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")

	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
