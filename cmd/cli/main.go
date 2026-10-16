// Package main is the entry point for the Flowbot CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	cmd, err := command.Execute(version.Buildtags)
	if err != nil {
		code := utils.ExitCode(err)
		if printErr := utils.PrintError(os.Stderr, utils.IsJSON(cmd), err); printErr != nil {
			if _, writeErr := fmt.Fprintf(os.Stderr, "Error: %v\n", err); writeErr != nil {
				os.Exit(code)
			}
		}
		os.Exit(code)
	}
}
