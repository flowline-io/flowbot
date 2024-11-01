package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/agent/updater"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
)

func checkUpdate() {
	flog.Info("Checking for updates...")
	needsUpdate, latest, err := updater.CheckUpdates()
	if err != nil {
		flog.Error(fmt.Errorf("Failed to check for updates, %w", err))
	} else if needsUpdate {
		flog.Info("New version available current %v latest %v", version.Buildtags, latest[1:])

		flog.Info("Updating to the latest version...")
		updated, err := updater.UpdateSelf()
		if !updated {
			flog.Info("Failed to update, error %v", err)
		} else {
			flog.Info("Updated successfully.")
			_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			os.Exit(0)
		}
	} else {
		flog.Info("You are using the latest version")
	}
}
