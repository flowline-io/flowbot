package command

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// IsJSON reports whether the command (or an ancestor) requested JSON output.
func IsJSON(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("output"); f != nil {
			return f.Value.String() == "json"
		}
	}
	return false
}

// PrintJSON writes v as indented JSON to stdout.
func PrintJSON(v any) error {
	data, err := sonic.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	_, _ = fmt.Println(string(data))
	return nil
}

// PrintEmptyList writes an empty JSON array when -o json is set, otherwise tableMsg.
func PrintEmptyList(cmd *cobra.Command, tableMsg string) error {
	if IsJSON(cmd) {
		return PrintJSON([]any{})
	}
	_, _ = fmt.Println(tableMsg)
	return nil
}

// PrintJSONError writes a protocol-shaped failed response as JSON to stdout.
// HTTP API responses stay client-safe via NewFailedResponse; local CLI errors
// surface err.Error() so operators see actionable diagnostics on the console.
func PrintJSONError(err error) {
	resp := protocol.NewFailedResponse(err)
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			resp.Message = apiErr.Message
		}
		if apiErr.RetCode != "" {
			resp.RetCode = apiErr.RetCode
		}
	} else if err != nil && (resp.Message == "" || resp.Message == "Unknown Error") {
		resp.Message = err.Error()
	}
	data, mErr := sonic.MarshalIndent(resp, "", "  ")
	if mErr != nil {
		_, _ = fmt.Printf("{\"status\":\"failed\",\"message\":%q}\n", err.Error())
		return
	}
	_, _ = fmt.Println(string(data))
}
