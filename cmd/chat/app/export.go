package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/client"
)

const exportFileExt = ".json"

// DefaultExportFilename builds the default export filename for a session id.
func DefaultExportFilename(sessionID string) string {
	safeID := sanitizeExportFilename(sessionID)
	if safeID == "" {
		safeID = "session"
	}
	return fmt.Sprintf("flowbot-chat-%s%s", safeID, exportFileExt)
}

// ResolveExportPath returns the destination path for /export.
// An empty args value writes to the default filename in the current directory.
func ResolveExportPath(args, sessionID string) (string, error) {
	path := strings.TrimSpace(args)
	if isSlashArgPlaceholder(path) {
		path = ""
	}
	if path == "" {
		path = DefaultExportFilename(sessionID)
	}
	if !strings.HasSuffix(strings.ToLower(path), exportFileExt) {
		path += exportFileExt
	}
	return filepath.Abs(path)
}

// isSlashArgPlaceholder reports autocomplete placeholder text that should not be used as a real argument.
func isSlashArgPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "<path>", "[path]":
		return true
	default:
		return false
	}
}

// WriteSessionExport serializes one server session export to a JSON file.
func WriteSessionExport(path string, export *client.ChatSessionExport) error {
	if export == nil {
		return fmt.Errorf("export payload is nil")
	}
	data, err := sonic.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal export: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write export: %w", err)
	}
	return nil
}

// FormatExportSuccess returns the transcript line shown after a successful export.
func FormatExportSuccess(path string, count int) string {
	return fmt.Sprintf("Exported %d entries to %s", count, path)
}

func sanitizeExportFilename(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			writeBuilder(&b, string(r))
		}
	}
	return b.String()
}
