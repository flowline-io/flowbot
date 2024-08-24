package types

type WorkflowMetadata struct {
	Name     string `json:"name" yaml:"name"`
	Describe string `json:"describe" yaml:"describe"`
	Triggers []struct {
		Type string `json:"type" yaml:"type"`
		Rule KV     `json:"rule,omitempty" yaml:"rule"`
	} `json:"triggers" yaml:"triggers"`
	Pipeline []string       `json:"pipeline" yaml:"pipeline"`
	Tasks    []WorkflowTask `json:"tasks" yaml:"tasks"`
}

type WorkflowTask struct {
	ID       string   `json:"id" yaml:"id"`
	Action   string   `json:"action" yaml:"action"`
	Describe string   `json:"describe,omitempty" yaml:"describe"`
	Params   KV       `json:"params,omitempty" yaml:"params"`
	Vars     []string `json:"vars,omitempty" yaml:"vars"`
	Conn     []string `json:"conn,omitempty" yaml:"conn"`
}
