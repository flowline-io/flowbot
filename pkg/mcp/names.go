package mcp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// AgentToolPrefix is the chat-agent tool name prefix for external MCP tools.
	AgentToolPrefix = "mcp_"
	// PermissionKeyPrefix is the permission key prefix for one MCP server.
	PermissionKeyPrefix = "mcp."
)

var serverNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidServerName reports whether name is a safe MCP server id (no underscores).
func ValidServerName(name string) bool {
	return serverNameRe.MatchString(name)
}

// AgentToolName builds mcp_<server>_<remote> for the chat agent registry.
func AgentToolName(server, remote string) string {
	return AgentToolPrefix + server + "_" + remote
}

// ServerFromAgentTool extracts the server id from an mcp_<server>_<remote> name.
func ServerFromAgentTool(tool string) (string, bool) {
	rest, ok := strings.CutPrefix(tool, AgentToolPrefix)
	if !ok {
		return "", false
	}
	server, _, ok := strings.Cut(rest, "_")
	if !ok || server == "" {
		return "", false
	}
	return server, true
}

// IsAgentTool reports whether name is an outbound MCP agent tool (mcp_<server>_*).
func IsAgentTool(name string) bool {
	_, ok := ServerFromAgentTool(name)
	return ok
}

// PermissionKey returns mcp.<server> for permission evaluation.
func PermissionKey(server string) string {
	return PermissionKeyPrefix + server
}

// ValidateServerName returns an error when name is empty or illegal.
func ValidateServerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("mcp server name is required")
	}
	if !ValidServerName(name) {
		return fmt.Errorf("mcp server name %q must match [a-z][a-z0-9-]*", name)
	}
	return nil
}
