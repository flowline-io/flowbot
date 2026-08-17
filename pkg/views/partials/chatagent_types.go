package partials

// SelectableModelOption is one model entry available for the session picker.
type SelectableModelOption struct {
	ID         string
	Name       string
	Multimodal bool
}

// WorkspaceOption is one first-level workspace directory for the composer picker.
type WorkspaceOption struct {
	Value string
	Label string
}

// ChatAgentEndpoints carries configurable HTTP paths for chat agent UI components.
type ChatAgentEndpoints struct {
	CreateURL          string
	ListURL            string
	DetailURLTemplate  string
	PinURLTemplate     string
	ArchiveURLTemplate string
	SettingsURL        string
	MessagesURL        string
	MediaURL           string
	CancelURL          string
	CloseURL           string
	ConfirmURL         string
	EventsURL          string
	InspectURL         string
	RenderMarkdownURL  string
	ContextURL         string
	TrajectoryURL      string
	TodosURL           string
	// SkillsURL returns enabled skills for the composer slash picker.
	SkillsURL string
	// Filter is the active session list filter query value.
	Filter string
	// PendingApprovalCount is how many sessions currently wait on tool approval.
	PendingApprovalCount int
	// SelectableModels is the list of models available in the composer/thread picker.
	SelectableModels []SelectableModelOption
	// DefaultModel is the global chat_model used when no session override is set.
	DefaultModel string
	// DefaultApprovalMode is the effective user/YAML approval mode for new sessions.
	DefaultApprovalMode string
	// WorkspaceOptions is the composer picker list (config root plus first-level dirs).
	WorkspaceOptions []WorkspaceOption
	// WorkspaceRootLabel is filepath.Base of chat_agent.workspace for empty-rel display.
	WorkspaceRootLabel string
}

// ChatAgentPendingConfirm is a tool approval still waiting on the active run.
type ChatAgentPendingConfirm struct {
	ID               string
	Tool             string
	Summary          string
	Permission       string
	Pattern          string
	SuggestedPattern string
	SuggestAlways    bool
}
