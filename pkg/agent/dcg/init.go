// Package dcg integrates Destructive Command Guard (dcg) as a pre-exec check
// for chat agent run_terminal and run_code tools.
package dcg

import (
	"fmt"
	"os/exec"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// Init installs the process-wide BinaryChecker using the embedded config.
// When dcg is missing from PATH it logs a warning and installs an ErrorChecker
// so the first guarded tool call fails closed.
func Init() {
	cfgPath, err := MaterializeConfig()
	if err != nil {
		flog.Warn("[dcg] materialize config failed: %v", err)
		SetDefaultChecker(ErrorChecker{Err: fmt.Errorf("dcg: materialize config: %w", err)})
		return
	}
	path, err := exec.LookPath("dcg")
	if err != nil {
		flog.Warn("[dcg] binary not found on PATH; run_terminal/run_code will fail closed until dcg is installed")
		SetDefaultChecker(ErrorChecker{Err: fmt.Errorf("dcg: binary not found on PATH: %w", err)})
		return
	}
	SetDefaultChecker(NewBinaryChecker(BinaryCheckerOptions{ConfigPath: cfgPath}))
	flog.Info("[dcg] initialized binary=%s config=%s", path, cfgPath)
}
