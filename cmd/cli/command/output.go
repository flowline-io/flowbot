package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

// IsJSON reports whether the command (or an ancestor) requested JSON output.
func IsJSON(cmd *cobra.Command) bool {
	return utils.IsJSON(cmd)
}

func render(cmd *cobra.Command, v any) error {
	return utils.Print(cmd, v, nil)
}

// PrintEmptyList writes {"data":[]} when -o json is set, otherwise tableMsg.
// Paginated commands keep their page by rendering the list result instead.
func PrintEmptyList(cmd *cobra.Command, tableMsg string) error {
	if IsJSON(cmd) {
		return utils.WriteEnvelope(cmd.OutOrStdout(), []any{}, nil, false)
	}
	return writef(cmd, "%s\n", tableMsg)
}

// stopAfterList reports whether the caller should return.
// JSON mode renders payload, including an empty page.
// Table mode prints tableMsg only when n is zero.
func stopAfterList(cmd *cobra.Command, tableMsg string, payload any, n int) (bool, error) {
	if IsJSON(cmd) {
		return true, render(cmd, payload)
	}
	if n == 0 {
		return true, writef(cmd, "%s\n", tableMsg)
	}
	return false, nil
}

func writef(cmd *cobra.Command, format string, args ...any) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
