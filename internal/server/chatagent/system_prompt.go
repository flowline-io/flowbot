package chatagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
)

const maxContextFileBytes = 32 * 1024

// ContextFile holds project-specific instructions injected into the system prompt.
type ContextFile struct {
	Path    string
	Content string
}

// BuildSystemPromptOptions configures system prompt construction.
type BuildSystemPromptOptions struct {
	// CustomPrompt replaces the default prompt body when non-empty.
	CustomPrompt string
	// SelectedTools limits which tools appear in the Available tools section.
	SelectedTools []string
	// ToolSnippets provides one-line descriptions keyed by tool name.
	ToolSnippets map[string]string
	// PromptGuidelines adds extra guideline bullets after tool-specific defaults.
	PromptGuidelines []string
	// AppendSystemPrompt is appended after the main body and before project context.
	AppendSystemPrompt string
	// CWD is the sandbox workspace root shown to the model.
	CWD string
	// ContextFiles are pre-loaded project instruction files.
	ContextFiles []ContextFile
	// Skills are agent skills injected into the prompt; nil loads from the database.
	Skills []Skill
	// Subagents are delegation targets injected into the prompt for the task tool.
	Subagents []Subagent
}

// DefaultToolSnippets returns one-line tool descriptions for the chat assistant.
func DefaultToolSnippets() map[string]string {
	return map[string]string{
		"run_terminal": "Run shell commands inside the workspace (ls, git, build, test, etc.)",
		"read_file":    "Read a text file from the workspace by relative path",
		"write_file":   "Write or overwrite a text file in the workspace, creating parent dirs as needed",
		"web_search":   "Search the web for documentation, APIs, or current facts",
		"run_code":     "Execute a code snippet (go, python, javascript, shell) in the workspace",
		"read_skill":   "Load full instructions for a named skill from the database",
		"task":         "Delegate a self-contained task to a specialized subagent that runs in isolation",
	}
}

// BuildSystemPrompt constructs the chat assistant system prompt.
func BuildSystemPrompt(options BuildSystemPromptOptions) string {
	cwd := normalizePromptPath(options.CWD)
	language := config.App.Flowbot.Language
	if language == "" {
		language = "English"
	}
	date := time.Now().UTC().Format("2006-01-02")

	tools := options.SelectedTools
	if len(tools) == 0 {
		tools = ActiveToolNames()
	}
	snippets := options.ToolSnippets
	if snippets == nil {
		snippets = DefaultToolSnippets()
	}

	appendSection := ""
	if text := strings.TrimSpace(options.AppendSystemPrompt); text != "" {
		appendSection = "\n\n" + text
	}

	contextFiles := options.ContextFiles
	if contextFiles == nil {
		contextFiles = LoadDefaultContextFiles(cwd)
	}

	skills := options.Skills
	subagents := options.Subagents

	if custom := strings.TrimSpace(options.CustomPrompt); custom != "" {
		return finalizePrompt(custom+appendSection, contextFiles, skills, subagents, date, cwd, language, tools)
	}

	toolsList := formatToolsList(tools, snippets)
	guidelines := formatGuidelines(tools, options.PromptGuidelines, language)

	body := defaultPromptIntro() + fmt.Sprintf(`

Available tools:
%s

In addition to the tools above, you may receive other custom tools depending on configuration.

Guidelines:
%s`, toolsList, guidelines)

	return finalizePrompt(body+appendSection, contextFiles, skills, subagents, date, cwd, language, tools)
}

// SystemPrompt builds the default chat assistant prompt from workspace, config, and DB skills.
func SystemPrompt(ctx context.Context, ws coding.Workspace) string {
	return CachedSystemPrompt(ctx, ws)
}

// defaultPromptIntro returns the role and agent-harness explanation for the default prompt.
func defaultPromptIntro() string {
	return `You are an expert assistant operating inside Flowbot, an agent harness. You help users with questions, research, planning, and hands-on work by reading files, executing commands, editing content, searching the web, and running code when needed.

Agent harness:
Flowbot wraps you in an Observe-Think-Act loop. Each user message starts a harness run: you reason, call tools when needed, receive tool results, and continue until you can answer without further tools or the step limit is reached. Conversation context and tool traces are persisted across turns so follow-up messages continue the same session.

Harness behavior you should expect:
- Tool calls run inside the workspace sandbox; paths outside the sandbox are rejected.
- Terminal and code execution have timeouts; long output may be truncated for context safety.
- Prefer tools over guessing file contents, command output, or current external facts.
- Make incremental changes when modifying files, verify with tools when practical, then summarize outcomes for the user.
- The user controls the session with chat commands: "chat" starts a session, "end" closes it.`
}

// LoadDefaultContextFiles discovers common project instruction files under cwd.
func LoadDefaultContextFiles(cwd string) []ContextFile {
	return loadContextFiles(cwd, nil)
}

func loadContextFiles(cwd string, explicit []string) []ContextFile {
	names := explicit
	if len(names) == 0 {
		names = []string{"AGENTS.md", "README.md"}
	}
	files := make([]ContextFile, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, name)
		}
		content, err := readContextFile(path)
		if err != nil {
			continue
		}
		displayPath := name
		if rel, err := filepath.Rel(cwd, path); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
		files = append(files, ContextFile{Path: normalizePromptPath(displayPath), Content: content})
	}
	return files
}

func readContextFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("context path is directory")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > maxContextFileBytes {
		data = data[:maxContextFileBytes]
		return string(data) + "\n...(truncated)", nil
	}
	return string(data), nil
}

func formatToolsList(tools []string, snippets map[string]string) string {
	lines := make([]string, 0, len(tools))
	for _, name := range tools {
		snippet, ok := snippets[name]
		if !ok || strings.TrimSpace(snippet) == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", name, snippet))
	}
	if len(lines) == 0 {
		return "(none)"
	}
	return strings.Join(lines, "\n")
}

func formatGuidelines(tools, extra []string, language string) string {
	set := make(map[string]struct{})
	list := make([]string, 0, 12)

	add := func(item string) {
		item = strings.TrimSpace(item)
		if item == "" {
			return
		}
		if _, exists := set[item]; exists {
			return
		}
		set[item] = struct{}{}
		list = append(list, item)
	}

	has := func(name string) bool {
		return slices.Contains(tools, name)
	}

	if has("run_terminal") && !has("web_search") {
		add("Use run_terminal for file listing and repository inspection (ls, find, git status)")
	}
	if has("read_file") {
		add("Read files with read_file before editing unfamiliar content")
	}
	if has("write_file") {
		add("Prefer minimal, focused edits; preserve existing style and conventions")
	}
	if has("web_search") {
		add("Use web_search for library docs or facts not present in the workspace")
	}
	if has("read_skill") {
		add("Use read_skill to load specialized instructions when a task matches an available skill")
	}
	if has("task") {
		add("Use the task tool to delegate a self-contained task to a matching subagent from available_subagents")
	}

	for _, item := range extra {
		add(item)
	}

	add("Be concise in your responses")
	add("Show file paths clearly when working with files")
	add("Never access paths outside the workspace sandbox")
	add(fmt.Sprintf("Answer in %s unless the user requests another language", language))

	lines := make([]string, len(list))
	for i, item := range list {
		lines[i] = "- " + item
	}
	return strings.Join(lines, "\n")
}

func finalizePrompt(
	body string,
	contextFiles []ContextFile,
	skills []Skill,
	subagents []Subagent,
	date, cwd, language string,
	tools []string,
) string {
	var prompt strings.Builder
	writePrompt(&prompt, body)

	if len(contextFiles) > 0 && hasTool(tools, "read_file") {
		writePrompt(&prompt, "\n\n<project_context>\n\nProject-specific instructions and guidelines:\n\n")
		for _, file := range contextFiles {
			writePrompt(&prompt, fmt.Sprintf("<project_instructions path=%q>\n%s\n</project_instructions>\n\n", file.Path, file.Content))
		}
		writePrompt(&prompt, "</project_context>\n")
	}

	if hasTool(tools, "read_skill") {
		writePrompt(&prompt, FormatSkillsForPrompt(skills))
	}

	if hasTool(tools, taskToolName) {
		writePrompt(&prompt, FormatSubagentsForPrompt(subagents))
	}

	writePrompt(&prompt, fmt.Sprintf("\nCurrent date: %s", date))
	writePrompt(&prompt, fmt.Sprintf("\nCurrent working directory: %s", cwd))
	writePrompt(&prompt, fmt.Sprintf("\nResponse language: %s", language))

	return prompt.String()
}

func writePrompt(b *strings.Builder, text string) {
	_, _ = b.WriteString(text)
}

func hasTool(tools []string, name string) bool {
	return slices.Contains(tools, name)
}

func normalizePromptPath(path string) string {
	return strings.ReplaceAll(filepath.Clean(path), "\\", "/")
}
