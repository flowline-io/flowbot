package rules

import (
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
	"github.com/rulego/rulego/api/types"
	"regexp"
	"strings"
)

// JSON

type JsonParser struct{}

func (p *JsonParser) DecodeRuleChain(rootRuleChain []byte) (types.RuleChain, error) {
	var def types.RuleChain
	err := sonic.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *JsonParser) DecodeRuleNode(rootRuleChain []byte) (types.RuleNode, error) {
	var def types.RuleNode
	err := sonic.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *JsonParser) EncodeRuleChain(def interface{}) ([]byte, error) {
	return sonic.MarshalIndent(def, "", "  ")
}

func (p *JsonParser) EncodeRuleNode(def interface{}) ([]byte, error) {
	return sonic.MarshalIndent(def, "", "  ")
}

// Yaml

type YamlParser struct{}

func (p *YamlParser) DecodeRuleChain(rootRuleChain []byte) (types.RuleChain, error) {
	var def types.RuleChain
	err := yaml.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *YamlParser) DecodeRuleNode(rootRuleChain []byte) (types.RuleNode, error) {
	var def types.RuleNode
	err := yaml.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *YamlParser) EncodeRuleChain(def interface{}) ([]byte, error) {
	return yaml.Marshal(def)
}

func (p *YamlParser) EncodeRuleNode(def interface{}) ([]byte, error) {
	return yaml.Marshal(def)
}

// DSL

type DslParser struct{}

func (p *DslParser) DecodeRuleChain(rootRuleChain []byte) (types.RuleChain, error) {
	var dsl RuleChain
	err := yaml.Unmarshal(rootRuleChain, &dsl)
	if err != nil {
		return types.RuleChain{}, fmt.Errorf("failed to unmarshal rule chain yaml: %w", err)
	}

	var connections []types.NodeConnection
	if len(dsl.Pipelines) > 0 {
		edges, nodes, err := parsePipelines(dsl.Pipelines)
		if err != nil {
			return types.RuleChain{}, fmt.Errorf("failed to parse pipelines: %w", err)
		}
		err = validateGraph(edges, nodes)
		if err != nil {
			return types.RuleChain{}, fmt.Errorf("failed to validate graph: %w", err)
		}

		for _, edge := range edges {
			connections = append(connections, types.NodeConnection{
				FromId: edge.From,
				ToId:   edge.To,
				Type:   edge.Type,
			})
		}
	} else {
		connections = dsl.Connections
	}

	return types.RuleChain{
		RuleChain: types.RuleChainBaseInfo{
			ID:             dsl.ID,
			Name:           dsl.Name,
			DebugMode:      dsl.DebugMode,
			Root:           dsl.Root,
			Disabled:       dsl.Disabled,
			Configuration:  dsl.Configuration,
			AdditionalInfo: dsl.AdditionalInfo,
		},
		Metadata: types.RuleMetadata{
			FirstNodeIndex: dsl.FirstNodeIndex,
			Endpoints:      dsl.Endpoints,
			Nodes:          dsl.Nodes,
			Connections:    connections,
		},
	}, err
}

func (p *DslParser) DecodeRuleNode(rootRuleChain []byte) (types.RuleNode, error) {
	var def types.RuleNode
	err := yaml.Unmarshal(rootRuleChain, &def)
	return def, err
}

func (p *DslParser) EncodeRuleChain(def interface{}) ([]byte, error) {
	return yaml.Marshal(def)
}

func (p *DslParser) EncodeRuleNode(def interface{}) ([]byte, error) {
	return yaml.Marshal(def)
}

func parsePipelines(lines []string) ([]Edge, map[string]bool, error) {
	var edges []Edge
	nodes := make(map[string]bool)
	// match arrow format: --any characters (including spaces)-->
	arrowRegex := regexp.MustCompile(`--([^-]+?)-->`)

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "pipelines:") {
			continue
		}

		// remove list mark "- " from line head
		line = strings.TrimPrefix(line, "- ")

		// find all arrow positions
		arrowMatches := arrowRegex.FindAllStringSubmatchIndex(line, -1)
		if len(arrowMatches) == 0 {
			return nil, nil, fmt.Errorf("line %d: no valid arrow found: %s", lineNum+1, line)
		}

		// extract first node (before first arrow)
		firstNodeEnd := arrowMatches[0][0]
		from := strings.TrimSpace(line[:firstNodeEnd])
		if from == "" {
			return nil, nil, fmt.Errorf("line %d: starting node is empty", lineNum+1)
		}
		nodes[from] = true
		prevNode := from

		// handle each arrow and its following node
		for i, match := range arrowMatches {
			// extract arrow type
			arrowType := line[match[2]:match[3]]
			arrowType = strings.TrimSpace(arrowType)
			if arrowType == "" {
				return nil, nil, fmt.Errorf("line %d: arrow type is empty", lineNum+1)
			}

			// extract node after arrow
			start := match[1] // arrow end position
			end := len(line)
			if i < len(arrowMatches)-1 {
				end = arrowMatches[i+1][0] // next arrow start position
			}
			to := strings.TrimSpace(line[start:end])
			if to == "" {
				return nil, nil, fmt.Errorf("line %d: target node is empty", lineNum+1)
			}

			// add edge
			edges = append(edges, Edge{
				From: prevNode,
				To:   to,
				Type: arrowType,
			})
			nodes[to] = true
			prevNode = to
		}
	}
	return edges, nodes, nil
}

func validateGraph(edges []Edge, allNodes map[string]bool) error {
	if len(edges) == 0 {
		return nil
	}

	// collect all connected nodes
	connectedNodes := make(map[string]bool)
	for _, edge := range edges {
		connectedNodes[edge.From] = true
		connectedNodes[edge.To] = true
	}

	// check if there are isolated nodes
	for node := range allNodes {
		if !connectedNodes[node] {
			return fmt.Errorf("found isolated node: %s", node)
		}
	}

	// check for duplicate connections
	connections := make(map[string]map[string]bool)
	for _, edge := range edges {
		if connections[edge.From] == nil {
			connections[edge.From] = make(map[string]bool)
		}
		if connections[edge.From][edge.To] {
			return fmt.Errorf("duplicate connection found: %s -> %s", edge.From, edge.To)
		}
		connections[edge.From][edge.To] = true
	}

	return nil
}
