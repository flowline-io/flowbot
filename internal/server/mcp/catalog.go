package mcp

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

const (
	kindCapability = "capability"
	kindPipeline   = "pipeline"
	kindWorkflow   = "workflow"
	kindFunction   = "function"
	kindHub        = "hub"
	kindAudit      = "audit"
	kindLineage    = "lineage"
	kindFlow       = "flow"
	kindEvents     = "events"
)

// ToolSpec is one MCP tool advertised on /mcp.
type ToolSpec struct {
	Name        string
	Group       string
	Operation   string
	Description string
	Mutation    bool
	Kind        string
	Input       []hub.ParamDef
	Scopes      []string
	Output      map[string]any
}

// BuildCatalog returns capability, pipeline/workflow/function, and hub tools
// after deny-list and mcp.include / mcp.exclude filters.
func BuildCatalog(descs []hub.Descriptor, include, exclude []string) []ToolSpec {
	var out []ToolSpec
	for _, desc := range descs {
		capName := string(desc.Type)
		for _, op := range desc.Operations {
			name := capName + "." + op.Name
			if pkgmcp.Denied(capName, op.Name) {
				continue
			}
			if !pkgmcp.Match(include, exclude, capName, name) {
				continue
			}
			mutation := capability.IsMutation(op.Name)
			out = append(out, ToolSpec{
				Name:        name,
				Group:       capName,
				Operation:   op.Name,
				Description: op.Description,
				Mutation:    mutation,
				Kind:        kindCapability,
				Input:       withPageParams(capName, op.Name, op.Input),
				Scopes:      expandScopes(op.Scopes, capName, mutation),
				Output:      pkgmcp.OutputSchema(capName, op.Name),
			})
		}
	}
	out = append(out, extraTools(include, exclude)...)
	return out
}

func extraTools(include, exclude []string) []ToolSpec {
	specs := []ToolSpec{
		{
			Name: "pipeline.list", Group: kindPipeline, Operation: "list", Kind: kindPipeline,
			Description: "List published pipelines for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Scopes:      []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.get", Group: kindPipeline, Operation: "get", Kind: kindPipeline,
			Description: "Get a published pipeline by name for the signed-in caller. Read-only.\nExample: {\"name\":\"example\"}",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Pipeline name"}},
			Scopes:      []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.run", Group: kindPipeline, Operation: "run", Kind: kindPipeline, Mutation: true,
			Description: "Start a manual run of an existing published pipeline for the signed-in caller. Creates or updates data.\nExample: {\"name\":\"example\"}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Pipeline name"},
				{Name: "event", Type: "object", Required: false, Description: "Optional event payload"},
			},
			Scopes: []string{auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.get_run", Group: kindPipeline, Operation: "get_run", Kind: kindPipeline,
			Description: "List recent runs for a pipeline, or one run when run_id is set for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Pipeline name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "workflow.list", Group: kindWorkflow, Operation: "list", Kind: kindWorkflow,
			Description: "List workflows for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Scopes:      []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.get", Group: kindWorkflow, Operation: "get", Kind: kindWorkflow,
			Description: "Get workflow metadata by name for the signed-in caller. Read-only.\nExample: {\"name\":\"example\"}",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Workflow name"}},
			Scopes:      []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.run", Group: kindWorkflow, Operation: "run", Kind: kindWorkflow, Mutation: true,
			Description: "Start a manual run of an existing workflow for the signed-in caller. Creates or updates data.\nExample: {\"name\":\"example\"}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Workflow name"},
				{Name: "input", Type: "object", Required: false, Description: "Optional workflow input"},
			},
			Scopes: []string{auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.get_run", Group: kindWorkflow, Operation: "get_run", Kind: kindWorkflow,
			Description: "List recent runs for a workflow, or one run when run_id is set for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Workflow name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "function.list", Group: kindFunction, Operation: "list", Kind: kindFunction,
			Description: "List published named functions for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Scopes:      []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "function.get", Group: kindFunction, Operation: "get", Kind: kindFunction,
			Description: "Get published function metadata without secrets for the signed-in caller. Read-only.\nExample: {\"name\":\"example\"}",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Function name"}},
			Scopes:      []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "function.run", Group: kindFunction, Operation: "run", Kind: kindFunction, Mutation: true,
			Description: "Invoke an existing published named function for the signed-in caller. Creates or updates data.\nExample: {\"name\":\"example\"}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Function name"},
				{Name: "version", Type: "number", Required: false, Description: "Published version; latest when omitted"},
				{Name: "event", Type: "object", Required: false, Description: "Event payload"},
			},
			Scopes: []string{auth.ScopeFunctionRun},
		},
		{
			Name: "function.get_run", Group: kindFunction, Operation: "get_run", Kind: kindFunction,
			Description: "List recent runs for a function, or one run when run_id is set for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Function name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "hub.apps", Group: kindHub, Operation: "apps", Kind: kindHub,
			Description: "List homelab apps known to the hub for the signed-in caller. Read-only.\nExample: {\"limit\":20}",
			Scopes:      []string{auth.ScopeHubAppsRead},
		},
		{
			Name: "hub.health", Group: kindHub, Operation: "health", Kind: kindHub,
			Description: "Hub capability and app health for the signed-in caller. Read-only.\nExample: {\"name\":\"example\"}",
			Scopes:      []string{auth.ScopeHubHealthRead},
		},
		{
			Name: "audit.list", Group: kindAudit, Operation: "list", Kind: kindAudit,
			Description: "List the caller's MCP and service audit rows. Read-only. admin:* may list every subject.\nExample: {\"action\":\"mcp.tool.call\",\"limit\":20}",
			Scopes:      []string{auth.ScopeAuditRead},
			Input: []hub.ParamDef{
				{Name: "action", Type: "string", Description: "Audit action, for example mcp.tool.call"},
				{Name: "subject_id", Type: "string", Description: "Subject id. Non-admins may only pass their own uid"},
				{Name: "target_type", Type: "string", Description: "Target type, for example mcp_tool"},
				{Name: "result", Type: "string", Description: "success, failure, or rejected", Enum: []string{"success", "failure", "rejected"}},
				{Name: "since", Type: "string", Description: "Inclusive RFC3339 lower bound"},
				{Name: "until", Type: "string", Description: "Inclusive RFC3339 upper bound"},
				{Name: "limit", Type: "int", Description: "Page size, default 20, maximum 100"},
				{Name: "cursor", Type: "string", Description: "Opaque cursor from the previous page"},
			},
		},
		{
			Name: "lineage.trace", Group: kindLineage, Operation: "trace", Kind: kindLineage,
			Description: "Multi-hop resource lineage from one seed (app+entity_id, event_id, or correlation_id). Read-only.\nExample: {\"app\":\"karakeep\",\"entity_id\":\"bm:991\",\"direction\":\"up\"}",
			Scopes:      []string{auth.ScopeLineageRead},
			Input: []hub.ParamDef{
				{Name: "app", Type: "string", Description: "Seed app when tracing by entity"},
				{Name: "entity_id", Type: "string", Description: "Seed entity_id when tracing by entity"},
				{Name: "event_id", Type: "string", Description: "Seed data_events event_id"},
				{Name: "correlation_id", Type: "string", Description: "Seed correlation_id across a run"},
				{Name: "direction", Type: "string", Description: "Walk direction", Enum: []string{"up", "down", "both"}},
				{Name: "depth", Type: "int", Description: "Max hop depth, default 10, maximum 10"},
			},
		},
		{
			Name: "lineage.search", Group: kindLineage, Operation: "search", Kind: kindLineage,
			Description: "Search distinct resource nodes by app, capability, or entity_id substring. Read-only.\nExample: {\"query\":\"bm:991\",\"limit\":20}",
			Scopes:      []string{auth.ScopeLineageRead},
			Input: []hub.ParamDef{
				{Name: "query", Type: "string", Required: true, Description: "Substring matched against app, capability, and entity_id"},
			},
		},
		{
			Name: "flow.graph", Group: kindFlow, Operation: "graph", Kind: kindFlow,
			Description: "Live-aggregated app topology from resource_links in a time window. Read-only.\nExample: {\"since\":\"2027-01-05T00:00:00Z\",\"app\":\"miniflux\"}",
			Scopes:      []string{auth.ScopeLineageRead},
			Input: []hub.ParamDef{
				{Name: "since", Type: "string", Description: "Inclusive RFC3339 lower bound; default 7d lookback"},
				{Name: "until", Type: "string", Description: "Inclusive RFC3339 upper bound; default now"},
				{Name: "app", Type: "string", Description: "Optional app filter on source or target"},
				{Name: "include_static", Type: "boolean", Description: "Accepted for compatibility; static edges are ignored until published"},
			},
		},
		{
			Name: "events.timeline", Group: kindEvents, Operation: "timeline", Kind: kindEvents,
			Description: "Paged data_events timeline. Payload data is omitted unless include_payload is true and the caller holds the matching service read scope.\nExample: {\"since\":\"2027-01-05T00:00:00Z\",\"limit\":20}",
			Scopes:      []string{auth.ScopeLineageRead},
			Input: []hub.ParamDef{
				{Name: "since", Type: "string", Description: "Inclusive RFC3339 lower bound; default 7d lookback"},
				{Name: "until", Type: "string", Description: "Inclusive RFC3339 upper bound; default now"},
				{Name: "app", Type: "string", Description: "Optional app filter"},
				{Name: "event_type", Type: "string", Description: "Optional event_type filter"},
				{Name: "correlation_id", Type: "string", Description: "Optional correlation_id filter"},
				{Name: "include_payload", Type: "boolean", Description: "Include event data when the caller has service read scope; default false"},
			},
		},
	}
	var out []ToolSpec
	for _, spec := range specs {
		if pkgmcp.Match(include, exclude, spec.Group, spec.Name) {
			spec.Input = withPageParams(spec.Group, spec.Operation, spec.Input)
			spec.Output = pkgmcp.OutputSchema(spec.Group, spec.Operation)
			switch spec.Name {
			case "audit.list", "lineage.search", "events.timeline":
				spec.Output = auditListOutputSchema()
			case "lineage.trace", "flow.graph":
				spec.Output = pkgmcp.OutputSchema("pipeline", "get")
			}
			out = append(out, spec)
		}
	}
	return out
}

func auditListOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{"type": "array"},
			"page":  map[string]any{"type": "object"},
		},
		"required": []string{"items", "page"},
	}
}

func withPageParams(group, op string, input []hub.ParamDef) []hub.ParamDef {
	if !pkgmcp.IsListTool(group, op) {
		return input
	}
	hasLimit := false
	hasCursor := false
	for i, p := range input {
		if p.Name == "sort_order" && len(p.Enum) == 0 {
			input[i].Enum = []string{"asc", "desc"}
		}
		if p.Name == "limit" {
			hasLimit = true
		}
		if p.Name == "cursor" {
			hasCursor = true
		}
	}
	if !hasLimit {
		input = append(input, hub.ParamDef{
			Name:        "limit",
			Type:        "int",
			Description: "Maximum items in this page. Defaults to 20 and cannot exceed 100.",
		})
	}
	if !hasCursor {
		input = append(input, hub.ParamDef{
			Name:        "cursor",
			Type:        "string",
			Description: "Opaque cursor from the previous page. Omit on the first page.",
		})
	}
	return input
}

func expandScopes(opScopes []string, capName string, mutation bool) []string {
	if len(opScopes) > 0 {
		out := append([]string{}, opScopes...)
		if !mutation {
			for _, s := range opScopes {
				if base, ok := strings.CutSuffix(s, ":read"); ok {
					out = append(out, base+":write", base+":run")
				}
			}
		}
		return out
	}
	if mutation {
		return []string{auth.ServiceScope(capName, "write")}
	}
	return []string{auth.ServiceScope(capName, "read"), auth.ServiceScope(capName, "write")}
}

// Allowed reports whether the token may call spec.
func (s ToolSpec) Allowed(scopes []string) bool {
	if auth.HasScope(scopes, auth.ScopeAdmin) {
		return true
	}
	for _, want := range s.Scopes {
		if auth.HasScope(scopes, want) {
			return true
		}
	}
	return false
}

// FilterByScopes keeps tools the token is allowed to call.
func FilterByScopes(specs []ToolSpec, scopes []string) []ToolSpec {
	out := make([]ToolSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Allowed(scopes) {
			out = append(out, spec)
		}
	}
	return out
}

func catalogFromConfig() []ToolSpec {
	return BuildCatalog(hub.Default.List(), config.App.MCP.Include, config.App.MCP.Exclude)
}
