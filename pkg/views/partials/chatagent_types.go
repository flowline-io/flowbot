package partials

// SelectableModelOption is one model entry available for the session picker.
type SelectableModelOption struct {
	ID   string
	Name string
}

// ChatAgentEndpoints carries configurable HTTP paths for chat agent UI components.
type ChatAgentEndpoints struct {
	CreateURL         string
	ListURL           string
	DetailURLTemplate string
	SettingsURL       string
	MessagesURL       string
	CancelURL         string
	CloseURL          string
	ConfirmURL        string
	EventsURL         string
	InspectURL        string
	RenderMarkdownURL string
	ContextURL        string
	TodosURL          string
	// SelectableModels is the list of models available in the composer/thread picker.
	SelectableModels []SelectableModelOption
	// DefaultModel is the global chat_model used when no session override is set.
	DefaultModel string
}
