package pages

// MCPToolPreview is one tool shown on the MCP settings page.
type MCPToolPreview struct {
	Name string
	Tier string
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
