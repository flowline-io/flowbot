package partials

// SelectableModelOption is one model entry available for the session picker.
type SelectableModelOption struct {
	ID         string
	Name       string
	Multimodal bool
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
	TodosURL           string
	// Filter is the active session list filter query value.
	Filter string
	// PendingApprovalCount is how many sessions currently wait on tool approval.
	PendingApprovalCount int
	// SelectableModels is the list of models available in the composer/thread picker.
	SelectableModels []SelectableModelOption
	// DefaultModel is the global chat_model used when no session override is set.
	DefaultModel string
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
