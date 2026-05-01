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
)

type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Operation struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Input       []ParamDef `json:"input,omitempty"`
	Output      []ParamDef `json:"output,omitempty"`
	Scopes      []string   `json:"scopes,omitempty"`
}

type Descriptor struct {
	Type        CapabilityType `json:"type"`
	Backend     string         `json:"backend"`
	App         string         `json:"app"`
	Description string         `json:"description,omitempty"`
	Operations  []Operation    `json:"operations,omitempty"`
	Instance    any            `json:"-"`
	Healthy     bool           `json:"healthy"`
}
