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
				Input:       op.Input,
				Scopes:      expandScopes(op.Scopes, capName, mutation),
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
			Description: "List published pipelines",
			Scopes:      []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.get", Group: kindPipeline, Operation: "get", Kind: kindPipeline,
			Description: "Get a published pipeline by name",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Pipeline name"}},
			Scopes:      []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.run", Group: kindPipeline, Operation: "run", Kind: kindPipeline, Mutation: true,
			Description: "Start a manual run of an existing published pipeline",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Pipeline name"},
				{Name: "event", Type: "object", Required: false, Description: "Optional event payload"},
			},
			Scopes: []string{auth.ScopePipelineRun},
		},
		{
			Name: "pipeline.get_run", Group: kindPipeline, Operation: "get_run", Kind: kindPipeline,
			Description: "List recent runs for a pipeline, or one run when run_id is set",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Pipeline name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopePipelineRead, auth.ScopePipelineRun},
		},
		{
			Name: "workflow.list", Group: kindWorkflow, Operation: "list", Kind: kindWorkflow,
			Description: "List workflows",
			Scopes:      []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.get", Group: kindWorkflow, Operation: "get", Kind: kindWorkflow,
			Description: "Get workflow metadata by name",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Workflow name"}},
			Scopes:      []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.run", Group: kindWorkflow, Operation: "run", Kind: kindWorkflow, Mutation: true,
			Description: "Start a manual run of an existing workflow",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Workflow name"},
				{Name: "input", Type: "object", Required: false, Description: "Optional workflow input"},
			},
			Scopes: []string{auth.ScopeWorkflowRun},
		},
		{
			Name: "workflow.get_run", Group: kindWorkflow, Operation: "get_run", Kind: kindWorkflow,
			Description: "List recent runs for a workflow, or one run when run_id is set",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Workflow name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopeWorkflowRead, auth.ScopeWorkflowRun},
		},
		{
			Name: "function.list", Group: kindFunction, Operation: "list", Kind: kindFunction,
			Description: "List published named functions",
			Scopes:      []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "function.get", Group: kindFunction, Operation: "get", Kind: kindFunction,
			Description: "Get published function metadata without secrets",
			Input:       []hub.ParamDef{{Name: "name", Type: "string", Required: true, Description: "Function name"}},
			Scopes:      []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "function.run", Group: kindFunction, Operation: "run", Kind: kindFunction, Mutation: true,
			Description: "Invoke an existing published named function",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Function name"},
				{Name: "version", Type: "number", Required: false, Description: "Published version; latest when omitted"},
				{Name: "event", Type: "object", Required: false, Description: "Event payload"},
			},
			Scopes: []string{auth.ScopeFunctionRun},
		},
		{
			Name: "function.get_run", Group: kindFunction, Operation: "get_run", Kind: kindFunction,
			Description: "List recent runs for a function, or one run when run_id is set",
			Input: []hub.ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Function name"},
				{Name: "run_id", Type: "number", Required: false, Description: "Optional run id"},
			},
			Scopes: []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun},
		},
		{
			Name: "hub.apps", Group: kindHub, Operation: "apps", Kind: kindHub,
			Description: "List homelab apps known to the hub",
			Scopes:      []string{auth.ScopeHubAppsRead},
		},
		{
			Name: "hub.health", Group: kindHub, Operation: "health", Kind: kindHub,
			Description: "Hub capability and app health",
			Scopes:      []string{auth.ScopeHubHealthRead},
		},
	}
	var out []ToolSpec
	for _, spec := range specs {
		if pkgmcp.Match(include, exclude, spec.Group, spec.Name) {
			out = append(out, spec)
		}
	}
	return out
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
