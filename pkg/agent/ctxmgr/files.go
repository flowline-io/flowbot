package ctxmgr

import (
	"slices"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

// FileOperations tracks workspace file activity for compaction summaries.
type FileOperations struct {
	Read    map[string]struct{}
	Written map[string]struct{}
	Edited  map[string]struct{}
}

// NewFileOperations creates empty file operation sets.
func NewFileOperations() FileOperations {
	return FileOperations{
		Read:    make(map[string]struct{}),
		Written: make(map[string]struct{}),
		Edited:  make(map[string]struct{}),
	}
}

// ExtractFileOpsFromMessage records file tool usage from an assistant message.
func ExtractFileOpsFromMessage(message msg.AgentMessage, ops FileOperations) {
	assistant, ok := message.(msg.AssistantMessage)
	if !ok {
		return
	}
	for _, call := range assistant.ToolCalls() {
		path := toolPath(call.Arguments)
		if path == "" {
			continue
		}
		switch call.Name {
		case "read_file", "read":
			ops.Read[path] = struct{}{}
		case "write_file", "write":
			ops.Written[path] = struct{}{}
		case "edit_file", "edit":
			ops.Edited[path] = struct{}{}
		}
	}
}

// ExtractFileOperations aggregates file ops from messages and prior compaction details.
func ExtractFileOperations(messages []msg.AgentMessage, entries []session.TreeEntry, prevCompactionIndex int) FileOperations {
	ops := NewFileOperations()
	if prevCompactionIndex >= 0 {
		prev := entries[prevCompactionIndex]
		mergeStoredFileOps(prev, ops)
	}
	for _, message := range messages {
		ExtractFileOpsFromMessage(message, ops)
	}
	return ops
}

func mergeStoredFileOps(entry session.TreeEntry, ops FileOperations) {
	for _, path := range entry.ReadFiles {
		ops.Read[path] = struct{}{}
	}
	for _, path := range entry.ModifiedFiles {
		ops.Edited[path] = struct{}{}
	}
}

// ComputeFileLists splits read-only and modified paths for summary output.
func ComputeFileLists(ops FileOperations) (readFiles, modifiedFiles []string) {
	modified := make(map[string]struct{})
	for path := range ops.Edited {
		modified[path] = struct{}{}
	}
	for path := range ops.Written {
		modified[path] = struct{}{}
	}
	for path := range ops.Read {
		if _, ok := modified[path]; !ok {
			readFiles = append(readFiles, path)
		}
	}
	for path := range modified {
		modifiedFiles = append(modifiedFiles, path)
	}
	slices.Sort(readFiles)
	slices.Sort(modifiedFiles)
	return readFiles, modifiedFiles
}

// FormatFileOperations renders file lists as XML blocks for summaries.
func FormatFileOperations(readFiles, modifiedFiles []string) string {
	sections := make([]string, 0, 2)
	if len(readFiles) > 0 {
		sections = append(sections, "<read-files>\n"+strings.Join(readFiles, "\n")+"\n</read-files>")
	}
	if len(modifiedFiles) > 0 {
		sections = append(sections, "<modified-files>\n"+strings.Join(modifiedFiles, "\n")+"\n</modified-files>")
	}
	if len(sections) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(sections, "\n\n")
}

func toolPath(arguments string) string {
	if strings.TrimSpace(arguments) == "" {
		return ""
	}
	var payload map[string]any
	if err := sonic.Unmarshal([]byte(arguments), &payload); err != nil {
		return ""
	}
	raw, ok := payload["path"]
	if !ok {
		return ""
	}
	path, ok := raw.(string)
	if !ok {
		return ""
	}
	return path
}
