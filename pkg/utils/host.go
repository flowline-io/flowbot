package utils

import (
	"github.com/shirou/gopsutil/v4/host"
)

func HostInfo() (hostID, hostname string, err error) {
	infoStat, err := host.Info()
	if err != nil {
		return "", "", err
	}
	return infoStat.HostID, infoStat.Hostname, nil
}
