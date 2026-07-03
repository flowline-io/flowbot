package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/subagent"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/tmc/langchaingo/llms"
)

const taskToolName = "task"

// TaskToolDeps carries the per-run metadata needed to delegate to a subagent.
type TaskToolDeps struct {
	// SessionID is the owning chat session, used to reuse permission gates.
	SessionID string
	// UID is the session owner, used to evaluate tool permissions in the subagent.
	UID types.Uid
	// Depth is the delegation depth of the caller (0 for the primary agent).
	Depth int
}

// TaskTool delegates a self-contained task to an isolated subagent loop.
type TaskTool struct {
	workspace coding.Workspace
	deps      TaskToolDeps
}

// NewTaskTool creates a task tool bound to a workspace and per-run delegation metadata.
func NewTaskTool(ws coding.Workspace, deps TaskToolDeps) TaskTool {
	return TaskTool{workspace: ws, deps: deps}
}

// Name returns the tool identifier.
func (TaskTool) Name() string { return taskToolName }

// Description explains the tool to the model.
func (TaskTool) Description() string {
	return "Delegates a self-contained task to a specialized subagent that runs in an isolated context and returns only its final result. Set subagent_type to a name from available_subagents."
}

// Parameters returns the JSON schema for tool arguments.
func (TaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subagent_type": map[string]any{
				"type":        "string",
				"description": "Subagent name from available_subagents",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short (3-5 word) summary of the delegated task",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "The detailed, self-contained task for the subagent",
			},
		},
		"required": []string{"subagent_type", "description", "prompt"},
	}
}

// Execute resolves the subagent definition, runs it in isolation, and returns the final result.
func (t TaskTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	subagentType := stringArg(args, "subagent_type")
	description := stringArg(args, "description")
	prompt := stringArg(args, "prompt")
	if subagentType == "" {
		return taskToolError(id, "subagent_type is required"), nil
	}
	if prompt == "" {
		return taskToolError(id, "prompt is required"), nil
	}

	def, err := subagentDefinitionFromStore(ctx, subagentType)
	if err != nil {
		flog.Warn("[chat-agent] task subagent lookup failed type=%s: %v", subagentType, err)
		return taskToolError(id, fmt.Sprintf("unknown subagent %q: %v", subagentType, err)), nil
	}

	systemPrompt, err := buildSubagentSystemPrompt(ctx, def)
	if err != nil {
		flog.Warn("[chat-agent] task subagent skills type=%s: %v", subagentType, err)
		return taskToolError(id, fmt.Sprintf("subagent skills: %v", err)), nil
	}

	taskRecord, err := beginSubagentTask(ctx, t.deps.SessionID, subagentType, description, prompt, t.deps.Depth+1)
	auditNote := ""
	if err != nil {
		flog.Warn("[chat-agent] task subagent record type=%s: %v", subagentType, err)
		auditNote = "\n\n[note: subagent task record could not be saved]"
	}

	model, err := t.resolveModel(ctx, def.Model)
	if err != nil {
		flog.Warn("[chat-agent] task subagent model type=%s: %v", subagentType, err)
		failSubagentTask(ctx, taskRecord, fmt.Sprintf("subagent model: %v", err))
		return taskToolError(id, fmt.Sprintf("subagent model: %v", err)), nil
	}

	childRegistry, err := NewSubagentRegistry(t.workspace, def.Skills)
	if err != nil {
		failSubagentTask(ctx, taskRecord, fmt.Sprintf("subagent registry: %v", err))
		return taskToolError(id, fmt.Sprintf("subagent registry: %v", err)), nil
	}
	if active := activeSubagentTools(def.Tools, def.Skills); len(active) > 0 {
		childRegistry.SetActive(active)
	}

	cfg, _, _, _, err := agentLoopConfig()
	if err != nil {
		failSubagentTask(ctx, taskRecord, fmt.Sprintf("subagent config: %v", err))
		return taskToolError(id, fmt.Sprintf("subagent config: %v", err)), nil
	}
	cfg.MaxSteps = subagentMaxSteps()

	hookRegistry := hooks.NewRegistry()
	RegisterHooks(hookRegistry, ChatHookDeps{
		SessionID:   t.deps.SessionID,
		UID:         t.deps.UID,
		SessionMode: LoadSessionMode(ctx, t.deps.SessionID),
	})
	cfg = hooks.BridgeConfig(ctx, hookRegistry, cfg)

	runDef := subagent.Definition{
		Name:         def.Name,
		Description:  def.Description,
		SystemPrompt: systemPrompt,
		Tools:        def.Tools,
		Skills:       def.Skills,
		Model:        def.Model,
	}
	result, runErr := subagent.Run(ctx, runDef, subagent.Deps{
		Model:    model,
		Registry: childRegistry,
		Config:   cfg,
		Depth:    t.deps.Depth + 1,
		MaxDepth: subagentMaxDepth(),
	}, prompt, func(update string) {
		if onUpdate == nil {
			return
		}
		_ = onUpdate(fmt.Sprintf("[%s] %s", subagentType, update))
	})
	if runErr != nil {
		flog.Warn("[chat-agent] task subagent run type=%s: %v", subagentType, runErr)
		failSubagentTask(ctx, taskRecord, runErr.Error())
		return taskToolError(id, fmt.Sprintf("subagent %q failed: %v", subagentType, runErr)), nil
	}

	text := strings.TrimSpace(result.Text)
	if text == "" {
		text = fmt.Sprintf("subagent %q completed with no output", subagentType)
	}
	if auditNote != "" {
		text += auditNote
	}
	completeSubagentTask(ctx, taskRecord, text)
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       taskToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

func (TaskTool) resolveModel(ctx context.Context, override string) (llms.Model, error) {
	name := strings.TrimSpace(override)
	if name == "" {
		name = strings.TrimSpace(config.App.ChatAgent.SubagentDefaultModel)
	}
	if name == "" {
		name = config.ChatAgentChatModel()
	}
	model, _, err := NewModelForTest(ctx, name)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func taskToolError(id, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       taskToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    true,
	}
}

func subagentMaxDepth() int {
	depth := config.App.ChatAgent.SubagentMaxDepth
	if depth <= 0 {
		return 1
	}
	return depth
}

func subagentMaxSteps() int {
	steps := config.App.ChatAgent.SubagentMaxSteps
	if steps <= 0 {
		return runMaxSteps()
	}
	return steps
}
