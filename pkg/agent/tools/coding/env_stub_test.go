package coding_test

import (
	"context"
	"os"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// failNthWriteEnv fails the Nth WriteFile call (1-based).
type failNthWriteEnv struct {
	env.OSExecutionEnv
	failAt int
	writes int
}

func (e *failNthWriteEnv) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	e.writes++
	if e.failAt > 0 && e.writes == e.failAt {
		return result.Err[struct{}, result.FileError](result.NewFileError("io_error", path, os.ErrPermission))
	}
	return e.OSExecutionEnv.WriteFile(ctx, path, data, perm)
}
