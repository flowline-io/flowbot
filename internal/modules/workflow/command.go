package workflow

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/n8n"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "workflow list",
		Help:   `List all workflows`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			workflows, err := client.ListWorkflows()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to list workflows: %v", err)}
			}

			if len(workflows) == 0 {
				return types.TextMsg{Text: "No workflows found"}
			}

			var parts []string
			parts = append(parts, fmt.Sprintf("*Workflows (%d)*", len(workflows)))
			for i, wf := range workflows {
				status := "Inactive"
				if wf.Active {
					status = "Active"
				}
				parts = append(parts, fmt.Sprintf("%d. %s [%s] - ID: %s", i+1, wf.Name, status, wf.ID))
			}

			return types.TextMsg{Text: strings.Join(parts, "\n")}
		},
	},
	{
		Define: "workflow get [id]",
		Help:   `Get workflow by ID`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow get <id>"}
			}

			id, _ := tokens[2].Value.String()
			if id == "" {
				return types.TextMsg{Text: "Workflow ID is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			workflow, err := client.GetWorkflow(id)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get workflow: %v", err)}
			}

			var parts []string
			parts = append(parts, fmt.Sprintf("*Workflow: %s*", workflow.Name))
			status := "Inactive"
			if workflow.Active {
				status = "Active"
			}
			parts = append(parts, fmt.Sprintf("Status: %s", status))
			parts = append(parts, fmt.Sprintf("ID: %s", workflow.ID))
			if workflow.TriggerCount > 0 {
				parts = append(parts, fmt.Sprintf("Trigger Count: %d", workflow.TriggerCount))
			}
			if len(workflow.Nodes) > 0 {
				parts = append(parts, fmt.Sprintf("Nodes: %d", len(workflow.Nodes)))
			}
			if len(workflow.Tags) > 0 {
				var tagNames []string
				for _, tag := range workflow.Tags {
					tagNames = append(tagNames, tag.Name)
				}
				parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(tagNames, ", ")))
			}

			return types.TextMsg{Text: strings.Join(parts, "\n")}
		},
	},
	{
		Define: "workflow create [name]",
		Help:   `Create a new workflow`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow create <name>"}
			}

			name, _ := tokens[2].Value.String()
			if name == "" {
				return types.TextMsg{Text: "Workflow name is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			workflow := &n8n.Workflow{
				Name:   name,
				Active: false,
				Nodes:  []n8n.Node{},
			}

			created, err := client.CreateWorkflow(workflow)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to create workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow created successfully!\nName: %s\nID: %s", created.Name, created.ID)}
		},
	},
	{
		Define: "workflow update [id] [name]",
		Help:   `Update workflow name`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 4 {
				return types.TextMsg{Text: "Usage: workflow update <id> <name>"}
			}

			id, _ := tokens[2].Value.String()
			name, _ := tokens[3].Value.String()
			if id == "" || name == "" {
				return types.TextMsg{Text: "Workflow ID and name are required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			// Get existing workflow first
			existing, err := client.GetWorkflow(id)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get workflow: %v", err)}
			}

			// Update name
			existing.Name = name
			updated, err := client.UpdateWorkflow(id, existing)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to update workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow updated successfully!\nName: %s\nID: %s", updated.Name, updated.ID)}
		},
	},
	{
		Define: "workflow delete [id]",
		Help:   `Delete a workflow`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow delete <id>"}
			}

			id, _ := tokens[2].Value.String()
			if id == "" {
				return types.TextMsg{Text: "Workflow ID is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			err = client.DeleteWorkflow(id)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to delete workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow %s deleted successfully", id)}
		},
	},
	{
		Define: "workflow activate [id]",
		Help:   `Activate a workflow`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow activate <id>"}
			}

			id, _ := tokens[2].Value.String()
			if id == "" {
				return types.TextMsg{Text: "Workflow ID is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			err = client.ActivateWorkflow(id)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to activate workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow %s activated successfully", id)}
		},
	},
	{
		Define: "workflow deactivate [id]",
		Help:   `Deactivate a workflow`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow deactivate <id>"}
			}

			id, _ := tokens[2].Value.String()
			if id == "" {
				return types.TextMsg{Text: "Workflow ID is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			err = client.DeactivateWorkflow(id)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to deactivate workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow %s deactivated successfully", id)}
		},
	},
	{
		Define: "workflow execute [id]",
		Help:   `Execute a workflow`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			if len(tokens) < 3 {
				return types.TextMsg{Text: "Usage: workflow execute <id>"}
			}

			id, _ := tokens[2].Value.String()
			if id == "" {
				return types.TextMsg{Text: "Workflow ID is required"}
			}

			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			err = client.ExecuteWorkflow(id, nil)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to execute workflow: %v", err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("Workflow %s executed successfully via webhook", id)}
		},
	},
	{
		Define: "workflow stat",
		Help:   `Get workflow statistics`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			client, err := n8n.GetClient()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to get n8n client: %v", err)}
			}

			workflows, err := client.ListWorkflows()
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to list workflows: %v", err)}
			}

			var activeCount, inactiveCount, totalTriggers int
			for _, wf := range workflows {
				if wf.Active {
					activeCount++
				} else {
					inactiveCount++
				}
				totalTriggers += wf.TriggerCount
			}

			var parts []string
			parts = append(parts, "*Workflow Statistics*")
			parts = append(parts, fmt.Sprintf("Total: %d", len(workflows)))
			parts = append(parts, fmt.Sprintf("Active: %d", activeCount))
			parts = append(parts, fmt.Sprintf("Inactive: %d", inactiveCount))
			parts = append(parts, fmt.Sprintf("Total Triggers: %d", totalTriggers))

			return types.TextMsg{Text: strings.Join(parts, "\n")}
		},
	},
}
