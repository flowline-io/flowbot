package n8n

import "time"

// Workflow represents an n8n workflow
type Workflow struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Active       bool           `json:"active,omitempty"`
	Nodes        []Node         `json:"nodes,omitempty"`
	Connections  map[string]any `json:"connections,omitempty"`
	CreatedAt    *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time     `json:"updatedAt,omitempty"`
	Settings     map[string]any `json:"settings,omitempty"`
	StaticData   map[string]any `json:"staticData,omitempty"`
	Tags         []Tag          `json:"tags,omitempty"`
	TriggerCount int            `json:"triggerCount,omitempty"`
}

// Node represents a node in an n8n workflow
type Node struct {
	ID               string         `json:"id,omitempty"`
	Name             string         `json:"name,omitempty"`
	Type             string         `json:"type,omitempty"`
	TypeVersion      float64        `json:"typeVersion,omitempty"`
	Position         []float64      `json:"position,omitempty"`
	Parameters       map[string]any `json:"parameters,omitempty"`
	Credentials      map[string]any `json:"credentials,omitempty"`
	Notes            string         `json:"notes,omitempty"`
	NotesInFlow      bool           `json:"notesInFlow,omitempty"`
	Disabled         bool           `json:"disabled,omitempty"`
	ContinueOnFail   bool           `json:"continueOnFail,omitempty"`
	AlwaysOutputData bool           `json:"alwaysOutputData,omitempty"`
	ExecuteOnce      bool           `json:"executeOnce,omitempty"`
	RetryOnFail      bool           `json:"retryOnFail,omitempty"`
	MaxTries         int            `json:"maxTries,omitempty"`
	WaitBetweenTries int            `json:"waitBetweenTries,omitempty"`
	WebhookID        string         `json:"webhookId,omitempty"`
}

// Tag represents a tag in n8n
type Tag struct {
	ID        string    `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
