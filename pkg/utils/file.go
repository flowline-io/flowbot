package utils

import "os"

func FileExist(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
