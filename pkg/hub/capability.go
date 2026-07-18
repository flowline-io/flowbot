package hub

// CapabilityType identifies a registered capability.
// For provider-backed capabilities this equals the provider ID (e.g. "karakeep").
type CapabilityType string

const (
	CapKarakeep     CapabilityType = "karakeep"
	CapMiniflux     CapabilityType = "miniflux"
	CapKanboard     CapabilityType = "kanboard"
	CapTrilium      CapabilityType = "trilium"
	CapMemos        CapabilityType = "memos"
	CapFireflyiii   CapabilityType = "fireflyiii"
	CapTransmission CapabilityType = "transmission"
	CapNocodb       CapabilityType = "nocodb"
	CapGitea        CapabilityType = "gitea"
	CapGithub       CapabilityType = "github"
	CapDevops       CapabilityType = "devops"
	CapExample      CapabilityType = "example"
	CapNotify       CapabilityType = "notify"
	CapAgent        CapabilityType = "agent"
	CapClip         CapabilityType = "clip"
)

// EventDef describes an event that a capability emits.
type EventDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitzero"`
}

// ParamDef describes a parameter for an operation.
type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitzero"`
	Required    bool   `json:"required,omitzero"`
}

// Operation describes a capability operation for discovery and auth scopes.
type Operation struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitzero"`
	Input       []ParamDef `json:"input,omitzero"`
	Output      []ParamDef `json:"output,omitzero"`
	Scopes      []string   `json:"scopes,omitzero"`
}

// Descriptor is the hub metadata for a registered capability.
type Descriptor struct {
	Type        CapabilityType `json:"type"`
	App         string         `json:"app"`
	Description string         `json:"description,omitzero"`
	Operations  []Operation    `json:"operations,omitzero"`
	Events      []EventDef     `json:"events,omitzero"`
	Instance    any            `json:"-"`
	Healthy     bool           `json:"healthy"`
}
