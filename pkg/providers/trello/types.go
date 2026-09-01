package trello

// Board is a Trello board resource.
type Board struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Desc   string `json:"desc,omitempty"`
	Closed bool   `json:"closed"`
	URL    string `json:"url,omitempty"`
}

// List is a Trello list on a board.
type List struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Closed  bool    `json:"closed"`
	IDBoard string  `json:"idBoard"`
	Pos     float64 `json:"pos,omitempty"`
}

// Card is a Trello card.
type Card struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Desc     string  `json:"desc,omitempty"`
	IDBoard  string  `json:"idBoard"`
	IDList   string  `json:"idList"`
	Pos      float64 `json:"pos,omitempty"`
	Closed   bool    `json:"closed"`
	URL      string  `json:"url,omitempty"`
	ShortURL string  `json:"shortUrl,omitempty"`
}

// Member is a Trello member (used for health checks).
type Member struct {
	ID       string `json:"id"`
	FullName string `json:"fullName,omitempty"`
	Username string `json:"username,omitempty"`
}

// Webhook is a registered Trello webhook.
type Webhook struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	IDModel     string `json:"idModel"`
	CallbackURL string `json:"callbackURL"`
	Active      bool   `json:"active"`
}

// SearchResult holds card matches from Trello search.
type SearchResult struct {
	Cards []Card `json:"cards"`
}

// WebhookAction is the action object inside a Trello webhook payload.
type WebhookAction struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// WebhookPayload is the top-level Trello webhook body.
type WebhookPayload struct {
	Action WebhookAction  `json:"action"`
	Model  map[string]any `json:"model"`
}
