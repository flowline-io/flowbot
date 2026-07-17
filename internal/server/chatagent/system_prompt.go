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
	// Mode selects plan vs normal prompt behavior; empty means normal.
	Mode string
}

// DefaultToolSnippets returns one-line tool descriptions for the chat assistant.
func DefaultToolSnippets() map[string]string {
	return map[string]string{
		"run_terminal":         "Run shell commands inside the workspace (git, build, test, etc.)",
		"list_dir":             "List files and directories under a workspace path",
		"glob_files":           "Find files by glob pattern (supports **); returns relative paths",
		"grep_files":           "Search file contents with a regular expression",
		"read_file":            "Read a text file from the workspace by relative path",
		"write_file":           "Write or overwrite a text file in the workspace, creating parent dirs as needed",
		"apply_patch":          "Apply an incremental multi-file patch (add/update/delete) inside the workspace",
		"web_search":           "Search the web for titles, URLs, and snippets",
		"web_fetch":            "Fetch text content from an http(s) URL (not localhost)",
		"run_code":             "Execute a Python or shell code snippet in the workspace",
		"read_skill":           "Load full skill instructions or an auxiliary file via optional path",
		"task":                 "Delegate a self-contained task to a specialized subagent that runs in isolation",
		scheduleToolName:       "Create a cron or one-shot scheduled agent task with name, prompt, and cron or run_at",
		updateScheduleToolName: "Update an existing scheduled task's cron, run_at, prompt, name, or state (active|paused)",
		listScheduleToolName:   "List active and paused scheduled tasks for the current user",
		cancelScheduleToolName: "Cancel a scheduled task by task_id",
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

	planSection := ""
	if options.Mode == ModePlan {
		planSection = planModePromptSection()
	}

	body := defaultPromptIntro() + planSection + fmt.Sprintf(`

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
	has := func(name string) bool { return slices.Contains(tools, name) }

	addCodingGuidelines(add, has)
	addProductGuidelines(add, has)
	for _, item := range extra {
		add(item)
	}
	add("Be concise in your responses")
	add("Show file paths clearly when working with files; reference workspace files as file://relative/path in markdown links")
	add("Never access paths outside the workspace sandbox")
	add(fmt.Sprintf("Answer in %s unless the user requests another language", language))

	lines := make([]string, len(list))
	for i, item := range list {
		lines[i] = "- " + item
	}
	return strings.Join(lines, "\n")
}

func addCodingGuidelines(add func(string), has func(string) bool) {
	if has("list_dir") {
		add("Use list_dir to inspect workspace directories")
	}
	if has("glob_files") {
		add("Use glob_files to find files by path pattern (for example **/*.go)")
	}
	if has("grep_files") {
		add("Use grep_files to search file contents with regular expressions")
	}
	if has("run_terminal") && !has("list_dir") && !has("web_search") {
		add("Use run_terminal for repository inspection (git status) and builds")
	}
	if has("read_file") {
		add("Read files with read_file before editing unfamiliar content")
	}
	if has("apply_patch") {
		add("Prefer apply_patch for incremental edits; use write_file for new files or full rewrites")
	}
	if has("write_file") {
		add("Prefer minimal, focused edits; preserve existing style and conventions")
	}
	if has("web_search") {
		add("Use web_search for library docs or facts not present in the workspace")
	}
	if has("web_fetch") {
		add("Use web_fetch to read a specific http(s) URL after you have a concrete link")
	}
}

func addProductGuidelines(add func(string), has func(string) bool) {
	if has("read_skill") {
		add("Use read_skill to load specialized instructions when a task matches an available skill")
	}
	if has("task") {
		add("Use the task tool to delegate a self-contained task to a matching subagent from available_subagents")
	}
	if has(scheduleToolName) {
		add("Use schedule_task for recurring (cron) or one-shot (run_at ISO8601 UTC) agent jobs; confirm the schedule with the user first")
		add("Scheduled tasks run in a separate session with the saved prompt; they do not continue the current conversation")
	}
	if has(updateScheduleToolName) {
		add("Use list_scheduled_tasks to find task_id before update_scheduled_task or cancel_scheduled_task")
		add("Use update_scheduled_task state=paused or state=active to pause and resume recurring or one-shot tasks")
	}
}

func planModePromptSection() string {
	return `

Plan mode:
You are in plan mode. Research thoroughly using read-only tools, then present a clear actionable plan.
Do not modify files, run shell commands, or execute code. Describe proposed changes step-by-step so the user can approve execution after exiting plan mode.
Your plan will be saved automatically; the server appends a plan:// link the user can open with /open.`
}

func planModeGuidelines() []string {
	return []string{
		"Plan mode is active: research and analyze thoroughly, then present a clear actionable plan",
		"Do not modify files, run shell commands, or execute code in plan mode",
		"Describe proposed changes step-by-step so the user can approve execution after exiting plan mode",
	}
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
