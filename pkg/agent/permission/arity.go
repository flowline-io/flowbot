package permission

import (
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
)

// arityTable maps command prefixes to how many tokens form the match prefix.
var arityTable = map[string]int{
	"git":            2,
	"npm":            2,
	"npm run":        3,
	"docker":         2,
	"docker compose": 3,
	"rm":             1,
	"grep":           1,
	"cd":             1,
	"cat":            1,
	"ls":             1,
	"go":             2,
	"cargo":          2,
}

// ParseBashCommand analyzes a shell command for permission matching.
type ParseBashCommand struct {
	// Prefix is the arity-derived command prefix used for rule matching.
	Prefix string
	// HasChain is true when pipes or command connectors are present.
	HasChain bool
	// Complex is true when parsing could not produce a reliable prefix.
	Complex bool
}

// AnalyzeBashCommand tokenizes command and extracts the permission prefix.
func AnalyzeBashCommand(command string) ParseBashCommand {
	command = strings.TrimSpace(command)
	if command == "" {
		return ParseBashCommand{Complex: true}
	}
	if hasShellChain(command) {
		first := strings.TrimSpace(splitFirstChain(command))
		parsed := analyzeSingleCommand(first)
		parsed.HasChain = true
		return parsed
	}
	return analyzeSingleCommand(command)
}

func hasShellChain(command string) bool {
	for _, sep := range []string{"|", "&&", "||", ";"} {
		if strings.Contains(command, sep) {
			return true
		}
	}
	return false
}

func splitFirstChain(command string) string {
	seps := []string{"|", "&&", "||", ";"}
	start := len(command)
	for _, sep := range seps {
		if idx := strings.Index(command, sep); idx >= 0 && idx < start {
			start = idx
		}
	}
	return command[:start]
}

func analyzeSingleCommand(command string) ParseBashCommand {
	words, err := shellwords.Parse(command)
	if err != nil || len(words) == 0 {
		return ParseBashCommand{Complex: true}
	}
	words = stripEnvAssignments(words)
	if len(words) == 0 {
		return ParseBashCommand{Complex: true}
	}
	words[0] = normalizeCommandToken(words[0])
	prefix := commandPrefix(words)
	if prefix == "" {
		return ParseBashCommand{Complex: true}
	}
	return ParseBashCommand{Prefix: prefix}
}

func stripEnvAssignments(words []string) []string {
	i := 0
	for i < len(words) && isEnvAssignment(words[i]) {
		i++
	}
	return words[i:]
}

func isEnvAssignment(token string) bool {
	if after, ok := strings.CutPrefix(token, "export "); ok {
		token = after
	}
	idx := strings.Index(token, "=")
	return idx > 0 && !strings.Contains(token[:idx], "/")
}

func normalizeCommandToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "./") || strings.HasPrefix(token, "../") {
		return filepath.Base(token)
	}
	if strings.Contains(token, "/") || strings.Contains(token, "\\") {
		return filepath.Base(token)
	}
	return token
}

func commandPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}
	bestKey := ""
	bestArity := 1
	for key, arity := range arityTable {
		if arity > len(words) {
			continue
		}
		candidate := strings.Join(words[:arity], " ")
		if strings.HasPrefix(candidate, key) && len(key) > len(bestKey) {
			bestKey = key
			bestArity = arity
		}
	}
	if bestKey != "" {
		return strings.Join(words[:bestArity], " ")
	}
	return words[0]
}
