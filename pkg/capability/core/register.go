package core

import (
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// serviceMarker is a non-nil instance used for hub registration.
type serviceMarker struct{}

// Register registers hub.CapCore with all core operations.
func Register() error {
	return capability.Register(buildSpec())
}

// CatalogSpec returns capability metadata for documentation (handlers may close over a nil service and must not be invoked).
func CatalogSpec() capability.Spec {
	return buildSpec()
}

func buildSpec() capability.Spec {
	return capability.Spec{
		Type:        hub.CapCore,
		Description: "Core runtime primitives: notify, clip, agent, HTTP, sandboxed exec, and KV",
		Instance:    serviceMarker{},
		Ops: []capability.OpDef{
			{
				Name: OpNotifySend, Description: "Send a notification using a template", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "template_id", Type: "string", Required: true, Description: "Template ID to render"},
					{Name: "channels", Type: "[]string", Required: true, Description: "Channels to send to"},
					{Name: "payload", Type: "map[string]any", Required: false, Description: "Template data payload"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID; pipeline injects Event.UID when omitted"},
				},
				Handler: notifySendInvoker,
			},
			{Name: OpNotifyHealth, Description: "Notify subsystem health", Handler: notifyHealthInvoker},
			{
				Name: OpClipCreate, Description: "Create a markdown clip and return its public URL", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "content", Type: "string", Required: true, Description: "Markdown body"},
					{Name: "created_by", Type: "string", Required: false, Description: "Optional creator identifier"},
				},
				Handler: clipCreateInvoker,
			},
			{
				Name: OpClipGet, Description: "Get a markdown clip by slug",
				Input:   []hub.ParamDef{{Name: "slug", Type: "string", Required: true, Description: "Clip slug"}},
				Handler: clipGetInvoker,
			},
			{Name: OpClipHealth, Description: "Clip subsystem health", Handler: clipHealthInvoker},
			{
				Name: OpAgentRun, Description: "Execute one autonomous agent turn with a prompt", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "prompt", Type: "string", Required: true, Description: "User prompt"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID"},
					{Name: "tools", Type: "[]string", Required: false, Description: "Tool allowlist"},
					{Name: "skills", Type: "[]string", Required: false, Description: "Skill allowlist"},
					{Name: "memory_scope", Type: "string", Required: false, Description: "Memory scope; defaults to pipeline name"},
				},
				Handler: agentRunInvoker,
			},
			{Name: OpAgentHealth, Description: "Agent subsystem health", Handler: agentHealthInvoker},
			{
				Name: OpHTTPRequest, Description: "Perform an outbound HTTP request", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "url", Type: "string", Required: true, Description: "Request URL"},
					{Name: "method", Type: "string", Required: false, Description: "HTTP method (default GET)"},
					{Name: "headers", Type: "map[string]any", Required: false, Description: "Request headers"},
					{Name: "body", Type: "string", Required: false, Description: "Request body"},
					{Name: "timeout_seconds", Type: "number", Required: false, Description: "Timeout in seconds"},
				},
				Handler: httpRequestInvoker,
			},
			{
				Name: OpRunTerminal, Description: "Run a shell command in the configured workspace sandbox", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "command", Type: "string", Required: true, Description: "Shell command"},
					{Name: "workdir", Type: "string", Required: false, Description: "Relative workdir under workspace"},
				},
				Handler: runTerminalInvoker,
			},
			{
				Name: OpRunCode, Description: "Run source code in the configured workspace sandbox", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "language", Type: "string", Required: true, Description: "python or shell"},
					{Name: "code", Type: "string", Required: true, Description: "Source code"},
					{Name: "filename", Type: "string", Required: false, Description: "Optional filename hint"},
					{Name: "workdir", Type: "string", Required: false, Description: "Relative workdir under workspace"},
				},
				Handler: runCodeInvoker,
			},
			{
				Name: OpKVGet, Description: "Get a persistent KV value",
				Input: []hub.ParamDef{
					{Name: "namespace", Type: "string", Required: true, Description: "Namespace (core/ prefix applied)"},
					{Name: "key", Type: "string", Required: true, Description: "Key"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID; defaults to instance scope"},
				},
				Handler: kvGetInvoker,
			},
			{
				Name: OpKVSet, Description: "Set a persistent KV value", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "namespace", Type: "string", Required: true, Description: "Namespace (core/ prefix applied)"},
					{Name: "key", Type: "string", Required: true, Description: "Key"},
					{Name: "value", Type: "any", Required: true, Description: "JSON-serializable value"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID; defaults to instance scope"},
					{Name: "ttl_seconds", Type: "number", Required: false, Description: "Optional TTL"},
				},
				Handler: kvSetInvoker,
			},
			{
				Name: OpKVDelete, Description: "Delete a persistent KV value", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "namespace", Type: "string", Required: true, Description: "Namespace (core/ prefix applied)"},
					{Name: "key", Type: "string", Required: true, Description: "Key"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID; defaults to instance scope"},
				},
				Handler: kvDeleteInvoker,
			},
		},
	}
}
