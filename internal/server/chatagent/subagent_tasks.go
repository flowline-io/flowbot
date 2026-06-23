package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

const (
	subagentTaskStatusRunning   = string(schema.AgentSubagentTaskStatusRunning)
	subagentTaskStatusCompleted = string(schema.AgentSubagentTaskStatusCompleted)
	subagentTaskStatusFailed    = string(schema.AgentSubagentTaskStatusFailed)
)

// subagentTaskRecord captures one delegated subagent task in storage.
type subagentTaskRecord struct {
	id int64
}

// beginSubagentTask persists a running task record before subagent execution starts.
func beginSubagentTask(ctx context.Context, sessionID, subagentName, description, prompt string, depth int) (*subagentTaskRecord, error) {
	if store.Database == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	task := &gen.AgentSubagentTask{
		SessionID:    sessionID,
		SubagentName: subagentName,
		Description:  description,
		Prompt:       prompt,
		Status:       subagentTaskStatusRunning,
		Depth:        depth,
		StartedAt:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Database.CreateAgentSubagentTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create subagent task: %w", err)
	}
	return &subagentTaskRecord{id: task.ID}, nil
}

// completeSubagentTask marks a task as completed with its final result text.
func completeSubagentTask(ctx context.Context, record *subagentTaskRecord, result string) {
	finishSubagentTask(ctx, record, subagentTaskStatusCompleted, result, "")
}

// failSubagentTask marks a task as failed with an error message.
func failSubagentTask(ctx context.Context, record *subagentTaskRecord, errText string) {
	finishSubagentTask(ctx, record, subagentTaskStatusFailed, "", errText)
}

func finishSubagentTask(ctx context.Context, record *subagentTaskRecord, status, result, errText string) {
	if record == nil || store.Database == nil {
		return
	}
	now := time.Now().UTC()
	task := &gen.AgentSubagentTask{
		ID:         record.id,
		Status:     status,
		Result:     strings.TrimSpace(result),
		ErrorText:  strings.TrimSpace(errText),
		FinishedAt: &now,
	}
	_ = store.Database.UpdateAgentSubagentTask(ctx, task)
}

// buildSubagentSystemPrompt appends filtered skills XML to the subagent system prompt.
func buildSubagentSystemPrompt(ctx context.Context, def subagentDefinition) (string, error) {
	prompt := def.SystemPrompt
	if len(def.Skills) == 0 {
		return prompt, nil
	}
	allSkills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		return "", err
	}
	filtered := FilterSkillsByNames(allSkills, def.Skills)
	if skillsPrompt := FormatSkillsForPrompt(filtered); skillsPrompt != "" {
		prompt += skillsPrompt
	}
	return prompt, nil
}

// subagentDefinition is the local view of a runnable subagent used by task tooling.
type subagentDefinition struct {
	Name         string
	Description  string
	SystemPrompt string
	Tools        []string
	Skills       []string
	Model        string
}

func subagentDefinitionFromStore(ctx context.Context, name string) (subagentDefinition, error) {
	def, err := GetSubagentDefinition(ctx, name)
	if err != nil {
		return subagentDefinition{}, err
	}
	return subagentDefinition{
		Name:         def.Name,
		Description:  def.Description,
		SystemPrompt: def.SystemPrompt,
		Tools:        def.Tools,
		Skills:       def.Skills,
		Model:        def.Model,
	}, nil
}

// activeSubagentTools returns the tool allowlist for a subagent registry, ensuring read_skill is active when skills are configured.
func activeSubagentTools(tools, skills []string) []string {
	if len(tools) == 0 && len(skills) == 0 {
		return nil
	}
	active := append([]string(nil), tools...)
	if len(skills) > 0 {
		active = appendUniqueTool(active, "read_skill")
	}
	return active
}

func appendUniqueTool(tools []string, name string) []string {
	for _, existing := range tools {
		if existing == name {
			return tools
		}
	}
	return append(tools, name)
}
