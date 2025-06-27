//go:build windows

package script

import (
	"fmt"
	"os"
)

func onceLock(id string) (*os.File, error) {
	return nil, fmt.Errorf("not support")
}
