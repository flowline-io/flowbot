package partials

// ChatAgentEndpoints carries configurable HTTP paths for chat agent UI components.
type ChatAgentEndpoints struct {
	CreateURL         string
	ListURL           string
	DetailURLTemplate string
	MessagesURL       string
	CancelURL         string
	CloseURL          string
	ConfirmURL        string
	EventsURL         string
	InspectURL        string
	RenderMarkdownURL string
	ContextURL        string
}
