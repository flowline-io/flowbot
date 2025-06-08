package rules

import (
	"fmt"
	"strings"
	"testing"
)

func generateYAML(edges []Edge) string {
	var builder strings.Builder
	_, _ = builder.WriteString("connections:\n")

	for _, edge := range edges {
		_, _ = builder.WriteString(fmt.Sprintf("  - fromId: %s\n", edge.From))
		_, _ = builder.WriteString(fmt.Sprintf("    toId: %s\n", edge.To))
		_, _ = builder.WriteString(fmt.Sprintf("    type: '%s'\n", edge.Type))
	}
	return builder.String()
}

func TestParsePipelines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     []Edge
		wantErr  bool
		errMatch string
	}{
		{
			name:  "Single Connection",
			input: "s1 --True--> s2",
			want: []Edge{
				{"s1", "s2", "True"},
			},
		},
		{
			name:  "Chained Connection",
			input: "A --T1--> B --T2--> C --T3--> D",
			want: []Edge{
				{"A", "B", "T1"},
				{"B", "C", "T2"},
				{"C", "D", "T3"},
			},
		},
		{
			name:  "Branch Connection",
			input: "Start --Main--> A --Path1--> B\nStart --Alt--> C --Path2--> D",
			want: []Edge{
				{"Start", "A", "Main"},
				{"A", "B", "Path1"},
				{"Start", "C", "Alt"},
				{"C", "D", "Path2"},
			},
		},
		{
			name:  "Connection with Spaces",
			input: "First Node --Complex Type--> Second Node --Another Type--> Final Node",
			want: []Edge{
				{"First Node", "Second Node", "Complex Type"},
				{"Second Node", "Final Node", "Another Type"},
			},
		},
		{
			name:  "YAML Format Input",
			input: "pipelines:\n- s1 --True--> s2\n- s2 --Success--> s3",
			want: []Edge{
				{"s1", "s2", "True"},
				{"s2", "s3", "Success"},
			},
		},
		{
			name:  "Mixed Chained and Independent Connection",
			input: "A --T1--> B\nB --T2--> C --T3--> D\nA --T4--> E",
			want: []Edge{
				{"A", "B", "T1"},
				{"B", "C", "T2"},
				{"C", "D", "T3"},
				{"A", "E", "T4"},
			},
		},
		{
			name:     "Empty Starting Node",
			input:    "--Type--> B",
			wantErr:  true,
			errMatch: "starting node is empty",
		},
		{
			name:     "Empty Arrow Type",
			input:    "A -- --> B",
			wantErr:  true,
			errMatch: "arrow type is empty",
		},
		{
			name:     "Invalid Arrow Format",
			input:    "A -> B",
			wantErr:  true,
			errMatch: "no valid arrow found",
		},
		{
			name:     "Empty Target Node",
			input:    "A --Type--> ",
			wantErr:  true,
			errMatch: "target node is empty",
		},
		{
			name:  "Complex Branch Structure",
			input: "Start --Init--> Step1 --Process--> Step2 --Verify--> End\nStart --Alt--> Parallel1 --Then--> Merge\nStep1 --Sub--> Parallel2 --Then--> Merge\nMerge --Finalize--> End",
			want: []Edge{
				{"Start", "Step1", "Init"},
				{"Step1", "Step2", "Process"},
				{"Step2", "End", "Verify"},
				{"Start", "Parallel1", "Alt"},
				{"Parallel1", "Merge", "Then"},
				{"Step1", "Parallel2", "Sub"},
				{"Parallel2", "Merge", "Then"},
				{"Merge", "End", "Finalize"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.input, "\n")
			edges, _, err := parsePipelines(lines)

			if (err != nil) != tt.wantErr {
				t.Errorf("parsePipelines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("Expected error containing %q, got %q", tt.errMatch, err.Error())
				}
				return
			}

			if len(edges) != len(tt.want) {
				t.Errorf("Expected %d edges, got %d", len(tt.want), len(edges))
				return
			}

			for i, edge := range edges {
				if edge != tt.want[i] {
					t.Errorf("Edge %d mismatch:\nGot:  %+v\nWant: %+v", i, edge, tt.want[i])
				}
			}
		})
	}
}

func TestValidateGraph(t *testing.T) {
	tests := []struct {
		name     string
		edges    []Edge
		nodes    map[string]bool
		wantErr  bool
		errMatch string
	}{
		{
			name: "Valid Connection",
			edges: []Edge{
				{"A", "B", "T1"},
				{"B", "C", "T2"},
			},
			nodes: map[string]bool{"A": true, "B": true, "C": true},
		},
		{
			name: "Isolated Node",
			edges: []Edge{
				{"A", "B", "T1"},
			},
			nodes: map[string]bool{
				"A": true, "B": true, "C": true, // C is isolated
			},
			wantErr:  true,
			errMatch: "found isolated node",
		},
		{
			name: "Duplicate Connection",
			edges: []Edge{
				{"A", "B", "T1"},
				{"A", "B", "T2"}, // duplicate A->B
			},
			nodes:    map[string]bool{"A": true, "B": true},
			wantErr:  true,
			errMatch: "duplicate connection found",
		},
		{
			name: "Complex Valid Graph",
			edges: []Edge{
				{"Start", "A", "T1"},
				{"Start", "B", "T2"},
				{"A", "C", "T3"},
				{"B", "C", "T4"},
				{"C", "End", "T5"},
			},
			nodes: map[string]bool{
				"Start": true, "A": true, "B": true, "C": true, "End": true,
			},
		},
		{
			name: "Self Loop",
			edges: []Edge{
				{"A", "A", "Loop"}, // self loop
			},
			nodes: map[string]bool{"A": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGraph(tt.edges, tt.nodes)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMatch) {
				t.Errorf("Expected error containing %q, got %q", tt.errMatch, err.Error())
			}
		})
	}
}

func TestGenerateYAML(t *testing.T) {
	tests := []struct {
		name  string
		edges []Edge
		want  string
	}{
		{
			name: "Basic Output",
			edges: []Edge{
				{"s1", "s2", "True"},
			},
			want: `connections:
  - fromId: s1
    toId: s2
    type: 'True'
`,
		},
		{
			name: "Multiple Connections Output",
			edges: []Edge{
				{"A", "B", "Type 1"},
				{"B", "C", "Type 2"},
				{"A", "D", "Type 3"},
			},
			want: `connections:
  - fromId: A
    toId: B
    type: 'Type 1'
  - fromId: B
    toId: C
    type: 'Type 2'
  - fromId: A
    toId: D
    type: 'Type 3'
`,
		},
		{
			name: "Special Characters Handling",
			edges: []Edge{
				{"Node-1", "Node_2", "Type:Special"},
			},
			want: `connections:
  - fromId: Node-1
    toId: Node_2
    type: 'Type:Special'
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateYAML(tt.edges)
			if got != tt.want {
				t.Errorf("generateYAML() mismatch:\nGot:\n%s\nWant:\n%s", got, tt.want)
			}
		})
	}
}

func TestEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
		errMatch string
	}{
		{
			name: "Basic Linear Conversion",
			input: `pipelines:
- s1 --True--> s2
- s2 --Success--> s3`,
			want: `connections:
  - fromId: s1
    toId: s2
    type: 'True'
  - fromId: s2
    toId: s3
    type: 'Success'
`,
		},
		{
			name: "Complex Linear Conversion",
			input: `pipelines:
- Start --Init--> Step1 --Process--> Step2 --Verify--> End
- Step1 --Alt--> Parallel --Then--> Merge
- Step2 --Another--> Merge --Finalize--> End`,
			want: `connections:
  - fromId: Start
    toId: Step1
    type: 'Init'
  - fromId: Step1
    toId: Step2
    type: 'Process'
  - fromId: Step2
    toId: End
    type: 'Verify'
  - fromId: Step1
    toId: Parallel
    type: 'Alt'
  - fromId: Parallel
    toId: Merge
    type: 'Then'
  - fromId: Step2
    toId: Merge
    type: 'Another'
  - fromId: Merge
    toId: End
    type: 'Finalize'
`,
		},
		{
			name: "Isolated Node Error",
			input: `pipelines:
- A --> B
- C --> D`,
			wantErr:  true,
			errMatch: "no valid arrow found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.input, "\n")
			edges, nodes, err := parsePipelines(lines)

			if !tt.wantErr && err != nil {
				t.Fatalf("parsePipelines failed: %v", err)
			}

			if !tt.wantErr {
				err = validateGraph(edges, nodes)
				if err != nil {
					t.Fatalf("validateGraph failed: %v", err)
				}

				got := generateYAML(edges)
				if got != tt.want {
					t.Errorf("Output mismatch:\nGot:\n%s\nWant:\n%s", got, tt.want)
				}
			} else {
				if err == nil {
					// If parsing succeeded, try to validate
					err = validateGraph(edges, nodes)
				}
				if err == nil || !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("Expected error containing %q, got %v", tt.errMatch, err)
				}
			}
		})
	}
}
