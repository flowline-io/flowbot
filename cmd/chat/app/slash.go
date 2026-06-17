package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxFileBytes   = 512 * 1024
	warnTokenLimit = 8000
)

// FileAttachment holds a local file queued for the next user message.
type FileAttachment struct {
	Path      string
	Content   string
	Truncated bool
	OrigSize  int
	EstTokens int
}

// ParseSlashCommand handles /commands entered in the input area.
func ParseSlashCommand(line string) (command string, args string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/") {
		return "", "", false
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", "", false
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	arg := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))
	return cmd, arg, true
}

// ReadLocalFile loads a file for /file attachment with size limits.
func ReadLocalFile(path string) (FileAttachment, error) {
	if path == "" {
		return FileAttachment{}, fmt.Errorf("file path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return FileAttachment{}, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return FileAttachment{}, err
	}
	origSize := len(data)
	truncated := origSize > maxFileBytes
	if truncated {
		data = data[:maxFileBytes]
	}
	content := string(data)
	est := estimateTokens(len(content))
	return FileAttachment{
		Path:      abs,
		Content:   content,
		Truncated: truncated,
		OrigSize:  origSize,
		EstTokens: est,
	}, nil
}

// FormatFileWarning returns a hint-line warning when a file was truncated or large.
func FormatFileWarning(att FileAttachment) string {
	if !att.Truncated && att.EstTokens <= warnTokenLimit {
		return ""
	}
	if att.Truncated {
		return fmt.Sprintf("[Warning] File too large, truncated to 512KB (est. ~%d tokens)", att.EstTokens)
	}
	return fmt.Sprintf("[Warning] Large attachment (est. ~%d tokens)", att.EstTokens)
}

// WrapUserMessage wraps optional file content around the user text.
func WrapUserMessage(att *FileAttachment, text string) string {
	if att == nil {
		return text
	}
	var b strings.Builder
	writeBuilder(&b, fmt.Sprintf("<file path=%q>\n%s\n</file>\n\n", att.Path, att.Content))
	writeBuilder(&b, text)
	return b.String()
}

func estimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	return chars / 4
}

// SlashHelp returns the /help text.
func SlashHelp() string {
	return strings.TrimSpace(`Commands:
  /new              Create a new session
  /end              Close current session
  /status           Show session info
  /context          Show context usage breakdown
  /resume           Reload saved session history
  /export [path]    Export current session to JSON
  /auth status      Show auth configuration
  /file <path>      Attach local file to next message
  /clear            Clear view (keep session)
  /help             Show this help
  /quit             Exit flowbot-chat`)
}
