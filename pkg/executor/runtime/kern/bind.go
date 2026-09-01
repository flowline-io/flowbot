// Package kern provides CLI-backed container execution for workflow and sandbox runners.
package kern

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ValidateBindMount checks whether a bind mount is permitted under the executor policy.
func ValidateBindMount(allowed bool, mnt types.Mount) error {
	if mnt.Type != types.MountTypeBind {
		return fmt.Errorf("kern runtime only supports bind mounts, got %q", mnt.Type)
	}
	if !allowed {
		return errors.New("bind mounts are not allowed")
	}
	if mnt.Source == "" {
		return errors.New("bind source is required")
	}
	if mnt.Target == "" {
		return errors.New("bind target is required")
	}
	return nil
}

// PrepareBindMount ensures the bind source exists on the host.
func PrepareBindMount(_ context.Context, mnt *types.Mount) error {
	if _, err := os.Stat(mnt.Source); os.IsNotExist(err) {
		if err := os.MkdirAll(mnt.Source, 0o750); err != nil {
			return fmt.Errorf("error creating bind mount directory %s: %w", mnt.Source, err)
		}
		flog.Info("Created bind mount: %s", mnt.Source)
	} else if err != nil {
		return fmt.Errorf("error stat bind mount %s: %w", mnt.Source, err)
	}
	return nil
}

// BindSpec formats a host:container bind for kern -v.
func BindSpec(source, target string, readOnly bool) string {
	if readOnly {
		return fmt.Sprintf("%s:%s:ro", source, target)
	}
	return fmt.Sprintf("%s:%s", source, target)
}
