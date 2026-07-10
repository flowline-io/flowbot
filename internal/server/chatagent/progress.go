package chatagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	// ProgressRelPath is the workspace-relative progress artifact path.
	ProgressRelPath = ".flowbot/progress.md"
	// ProgressTokenCap is the hard token limit for progress injected into context.
	ProgressTokenCap   = 500
	progressCustomType = "progress"
)

// ProgressArtifact holds the durable goal/done/next summary for a workspace run.
type ProgressArtifact struct {
	// Goal is the primary user objective.
	Goal string
	// Done lists completed steps.
	Done []string
	// Next lists remaining steps.
	Next []string
}

// ProgressFilePath returns the absolute path to the progress artifact.
func ProgressFilePath(workspaceRoot string) string {
	return filepath.Join(strings.TrimSpace(workspaceRoot), ProgressRelPath)
}

// FormatProgressMarkdown renders a progress artifact as markdown.
func FormatProgressMarkdown(p ProgressArtifact) string {
	var b strings.Builder
	_, _ = b.WriteString("# Progress\n\n")
	_, _ = b.WriteString("## Goal\n")
	goal := strings.TrimSpace(p.Goal)
	if goal == "" {
		goal = "(none)"
	}
	_, _ = b.WriteString(goal)
	_, _ = b.WriteString("\n\n## Done\n")
	writeBulletList(&b, p.Done)
	_, _ = b.WriteString("\n## Next\n")
	writeBulletList(&b, p.Next)
	_, _ = b.WriteString("\n")
	return b.String()
}

// ParseProgressMarkdown parses a progress.md document into an artifact.
func ParseProgressMarkdown(text string) ProgressArtifact {
	p := ProgressArtifact{}
	section := ""
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.EqualFold(trimmed, "## Goal"):
			section = "goal"
		case strings.EqualFold(trimmed, "## Done"):
			section = "done"
		case strings.EqualFold(trimmed, "## Next"):
			section = "next"
		case strings.HasPrefix(trimmed, "#"):
			section = ""
		case section == "goal" && trimmed != "" && trimmed != "(none)":
			if p.Goal == "" {
				p.Goal = trimmed
			} else {
				p.Goal += " " + trimmed
			}
		case section == "done":
			if item, ok := bulletItem(trimmed); ok {
				p.Done = append(p.Done, item)
			}
		case section == "next":
			if item, ok := bulletItem(trimmed); ok {
				p.Next = append(p.Next, item)
			}
		}
	}
	return p
}

// LoadProgress reads the progress artifact from the workspace when present.
func LoadProgress(workspaceRoot string) (ProgressArtifact, error) {
	path := ProgressFilePath(workspaceRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProgressArtifact{}, nil
		}
		return ProgressArtifact{}, err
	}
	return ParseProgressMarkdown(string(data)), nil
}

// SaveProgress writes the progress artifact under the workspace.
func SaveProgress(workspaceRoot string, p ProgressArtifact) error {
	path := ProgressFilePath(workspaceRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(FormatProgressMarkdown(p)), 0o644)
}

// TruncateProgressSummary caps text to approximately maxTokens using EstimateTextTokens.
func TruncateProgressSummary(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	if EstimateTextTokens(text) <= maxTokens {
		return text
	}
	// char/4 is a conservative lower bound used while binary-searching a cut point
	low, high := 0, len(text)
	for low < high {
		mid := (low + high + 1) / 2
		if EstimateTextTokens(text[:mid]) <= maxTokens {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low <= 0 {
		approx := min(maxTokens*4, len(text))
		return text[:approx]
	}
	return text[:low]
}

// DeriveProgressFromMessages builds a progress artifact from the working message list.
func DeriveProgressFromMessages(messages []msg.AgentMessage) ProgressArtifact {
	p := ProgressArtifact{}
	for _, item := range messages {
		switch m := item.(type) {
		case msg.UserMessage:
			text := strings.TrimSpace(textFromParts(m.Parts))
			if text == "" {
				continue
			}
			if p.Goal == "" {
				p.Goal = truncateRunes(text, 240)
			}
		case msg.ToolResultMessage:
			label := m.Name
			if m.IsError {
				label += " (error)"
			} else {
				label += " (ok)"
			}
			p.Done = appendUnique(p.Done, label)
		case msg.AssistantMessage:
			text := strings.TrimSpace(m.TextContent())
			if text == "" {
				continue
			}
			if len(m.ToolCalls()) == 0 {
				p.Next = []string{"awaiting next user input"}
			} else {
				names := make([]string, 0, len(m.ToolCalls()))
				for _, call := range m.ToolCalls() {
					names = append(names, call.Name)
				}
				p.Next = []string{fmt.Sprintf("continue after tools: %s", strings.Join(names, ", "))}
			}
		}
	}
	if len(p.Next) == 0 && p.Goal != "" {
		p.Next = []string{"continue working toward the goal"}
	}
	if len(p.Done) > 12 {
		p.Done = p.Done[len(p.Done)-12:]
	}
	return p
}

// registerProgressHooks maintains and injects the workspace progress artifact.
func registerProgressHooks(reg *hooks.Registry) {
	workspaceRoot := strings.TrimSpace(config.App.ChatAgent.Workspace)
	if workspaceRoot == "" {
		return
	}

	hooks.OnContext(reg, func(_ context.Context, event hooks.ContextEvent) (*hooks.ContextResult, error) {
		derived := DeriveProgressFromMessages(event.Messages)
		existing, err := LoadProgress(workspaceRoot)
		if err != nil {
			flog.Debug("[chat-agent] load progress: %v", err)
		} else if existing.Goal != "" && derived.Goal == "" {
			derived.Goal = existing.Goal
		}
		if err := SaveProgress(workspaceRoot, derived); err != nil {
			flog.Debug("[chat-agent] save progress: %v", err)
		}

		summary := TruncateProgressSummary(FormatProgressMarkdown(derived), ProgressTokenCap)
		if strings.TrimSpace(summary) == "" {
			return nil, nil
		}
		injected := msg.CustomMessage{
			CustomType: progressCustomType,
			Parts:      []msg.ContentPart{msg.TextPart{Text: summary}},
		}
		out := make([]msg.AgentMessage, 0, len(event.Messages)+1)
		out = append(out, stripProgressMessages(event.Messages)...)
		// Keep progress near the front so compaction still sees the durable summary.
		if len(out) == 0 {
			out = append(out, injected)
		} else {
			out = append([]msg.AgentMessage{out[0], injected}, out[1:]...)
		}
		return &hooks.ContextResult{Messages: out}, nil
	})

	hooks.OnObservation(reg, hooks.EventSavePoint, func(_ context.Context, _ hooks.ObservationEvent) error {
		path := ProgressFilePath(workspaceRoot)
		if _, err := os.Stat(path); err != nil {
			return nil
		}
		flog.Debug("[chat-agent] progress artifact ready path=%s", path)
		return nil
	})
}

func stripProgressMessages(messages []msg.AgentMessage) []msg.AgentMessage {
	out := make([]msg.AgentMessage, 0, len(messages))
	for _, item := range messages {
		if custom, ok := item.(msg.CustomMessage); ok && custom.CustomType == progressCustomType {
			continue
		}
		out = append(out, item)
	}
	return out
}

func writeBulletList(b *strings.Builder, items []string) {
	if len(items) == 0 {
		_, _ = b.WriteString("- (none)\n")
		return
	}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		_, _ = b.WriteString("- ")
		_, _ = b.WriteString(item)
		_, _ = b.WriteString("\n")
	}
}

func bulletItem(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "- "):
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if item == "" || item == "(none)" {
			return "", false
		}
		return item, true
	case strings.HasPrefix(trimmed, "* "):
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "* "))
		if item == "" || item == "(none)" {
			return "", false
		}
		return item, true
	default:
		return "", false
	}
}

func appendUnique(items []string, value string) []string {
	if slices.Contains(items, value) {
		return items
	}
	return append(items, value)
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
