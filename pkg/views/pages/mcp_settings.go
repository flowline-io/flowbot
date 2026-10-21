package pages

// MCPToolPreview is one tool shown on the MCP settings page.
type MCPToolPreview struct {
	Name string
	Tier string
	// Kind is read, write, or destructive. It selects the tier chip.
	Kind string
}

func mcpTierChipClass(kind string) string {
	switch kind {
	case "write":
		return "flowbot-chip flowbot-chip-warning"
	case "destructive":
		return "flowbot-chip flowbot-chip-error"
	default:
		return "flowbot-chip flowbot-chip-muted"
	}
}

// MCPSettingsData is the MCP settings page.
type MCPSettingsData struct {
	Enabled   bool
	URL       string
	Preset    string
	Scopes    []string
	Custom    []MCPScopeChoice
	Tools     []MCPToolPreview
	Snippets  []MCPSnippet
	Token     string
	CSRFToken string
	Error     string
}

// MCPScopeChoice is one checkbox on the custom preset.
type MCPScopeChoice struct {
	Value   string
	Checked bool
}

// MCPSnippet is one client tab.
type MCPSnippet struct {
	Client  string
	Target  string
	Snippet string
}
