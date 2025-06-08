package rules

import "github.com/rulego/rulego/api/types"

// RuleChain defines a rule chain.
type RuleChain struct {
	// RuleChainBaseInfo

	// ID is the unique identifier of the rule chain.
	ID string `json:"id" yaml:"id"`
	// Name is the name of the rule chain.
	Name string `json:"name" yaml:"name"`
	// DebugMode indicates whether the node is in debug mode. If true, a debug callback function is triggered when the node processes messages.
	// This setting overrides the `DebugMode` configuration of the node.
	DebugMode bool `json:"debugMode" yaml:"debugMode"`
	// Root indicates whether this rule chain is a root or a sub-rule chain. (Used only as a marker, not applied in actual logic)
	Root bool `json:"root" yaml:"root"`
	// Disabled indicates whether the rule chain is disabled.
	Disabled bool `json:"disabled" yaml:"disabled"`
	// Configuration contains the configuration information of the rule chain.
	Configuration types.Configuration `json:"configuration,omitempty" yaml:"configuration"`
	// AdditionalInfo is an extension field.
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty" yaml:"additionalInfo"`

	// RuleMetadata

	// FirstNodeIndex is the index of the first node in data flow, default is 0.
	FirstNodeIndex int `json:"firstNodeIndex" yaml:"firstNodeIndex"`
	// Nodes are the component definitions of the nodes.
	Endpoints []*types.EndpointDsl `json:"endpoints,omitempty" yaml:"endpoints"`
	// Nodes are the component definitions of the nodes.
	// Each object represents a rule node within the rule chain.
	Nodes []*types.RuleNode `json:"nodes" yaml:"nodes"`
	// Connections define the connections between two nodes in the rule chain.
	Connections []types.NodeConnection `json:"connections" yaml:"connections"`
	// Deprecated: Use Flow Node instead.
	// RuleChainConnections are the connections between a node and a sub-rule chain.
	RuleChainConnections []types.RuleChainConnection `json:"ruleChainConnections,omitempty" yaml:"ruleChainConnections"`

	// Custom fields
	// Pipelines is node pipelines, if set pipelines will be used, otherwise will use connections
	Pipelines []string `json:"pipelines" yaml:"pipelines"`
}

type Edge struct {
	From string
	To   string
	Type string
}
