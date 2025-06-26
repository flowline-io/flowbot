package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/agent/updater"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/version"
	"github.com/shirou/gopsutil/v4/host"
)

func checkUpdate() {
	flog.Info("[updater] Checking for updates...")
	needsUpdate, latest, err := updater.CheckUpdates()
	if err != nil {
		flog.Error(fmt.Errorf("[updater] failed to check for updates, %w", err))
	} else if needsUpdate {
		flog.Info("[updater] New version available current %v latest %v", version.Buildtags, latest[1:])

		flog.Info("[updater] Updating to the latest version...")
		updated, err := updater.UpdateSelf()
		if !updated {
			flog.Info("[updater] Failed to update, error %v", err)
		} else {
			flog.Info("[updater] Updated successfully.")
			_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			os.Exit(0)
		}
	} else {
		flog.Info("[updater] Currently are using the latest version")
	}
}

func hostInfo() (string, string) {
	infoStat, err := host.Info()
	if err != nil {
		flog.Error(err)
		return "", ""
	}
	flog.Info("host info: %s %s", infoStat.HostID, infoStat.Hostname)
	return infoStat.HostID, infoStat.Hostname
}
