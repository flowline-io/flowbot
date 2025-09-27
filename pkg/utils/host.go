package utils

import (
	"github.com/shirou/gopsutil/v4/host"
)

func HostInfo() (string, string, error) {
	infoStat, err := host.Info()
	if err != nil {
		return "", "", err
	}
	return infoStat.HostID, infoStat.Hostname, nil
}
