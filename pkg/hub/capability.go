package hub

type CapabilityType string

const (
	CapBookmark     CapabilityType = "bookmark"
	CapArchive      CapabilityType = "archive"
	CapReader       CapabilityType = "reader"
	CapKanban       CapabilityType = "kanban"
	CapFinance      CapabilityType = "finance"
	CapInfra        CapabilityType = "infra"
	CapShellHistory CapabilityType = "shell_history"
	CapNotify       CapabilityType = "notify"
)

type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitzero"`
	Required    bool   `json:"required,omitzero"`
}

type Operation struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitzero"`
	Input       []ParamDef `json:"input,omitzero"`
	Output      []ParamDef `json:"output,omitzero"`
	Scopes      []string   `json:"scopes,omitzero"`
}

type Descriptor struct {
	Type        CapabilityType `json:"type"`
	Backend     string         `json:"backend"`
	App         string         `json:"app"`
	Description string         `json:"description,omitzero"`
	Operations  []Operation    `json:"operations,omitzero"`
	Instance    any            `json:"-"`
	Healthy     bool           `json:"healthy"`
}
