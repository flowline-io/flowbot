package workflow

import (
	"reflect"
	"testing"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
)

const okYaml = `---
name: example
describe: do something...

triggers:
  - type: manual # cron, manual, webhook
  - type: cron
    rule:
      spec: '* * * * *' # if cron


pipeline:
  - input -> add_two_number -> out1
  - add_two_number -> out2


tasks:
  - id: input
    action: in_workflow_action@dev
    describe: do something... # optional
    params: # optional
      param1: val1
      param2: val2
    vars: # optional
      - var1
      - var2
    conn: # optional
      - conn1
      - conn2

  - id: add_two_number
    action: add_workflow_action@dev

  - id: out1
    action: out_workflow_action@dev

  - id: out2
    action: out_workflow_action@dev`

const failYaml = `---
name: example
describe: do something...

triggers:
  - type: manual # cron, manual, webhook
  - type: cron
    rule:
      spec: '* * * * *' # if cron


pipeline:
  - input -> add_two_number -> out1
  - add_two_number -> out2


tasks:
  - id: input
    action: in_workflow_action
    describe: do something... # optional
    params: # optional
      param1: val1
      param2: val2
    vars: # optional
      - var1
      - var2
    conn: # optional
      - conn1
      - conn2`

func TestParseYamlWorkflow(t *testing.T) {
	type args struct {
		code string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.Workflow
		want1   []*model.WorkflowTrigger
		want2   *model.Dag
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				code: okYaml,
			},
			want: &model.Workflow{
				Name:     "example",
				Describe: "do something...",
				State:    model.WorkflowEnable,
			},
			want1: []*model.WorkflowTrigger{
				{
					Type:  model.TriggerManual,
					State: model.WorkflowTriggerEnable,
				},
				{
					Type: model.TriggerCron,
					Rule: model.JSON{
						"spec": "* * * * *",
					},
					State: model.WorkflowTriggerEnable,
				},
			},
			want2: &model.Dag{
				Nodes: []*model.Node{
					{
						Id:       "input",
						Describe: "do something...",
						Bot:      "dev",
						RuleId:   "in_workflow_action",
						Parameters: map[string]interface{}{
							"param1": "val1",
							"param2": "val2",
						},
						Variables:   []string{"var1", "var2"},
						Connections: []string{"conn1", "conn2"},
						Status:      model.NodeDefault,
					},
					{
						Id:     "add_two_number",
						Bot:    "dev",
						RuleId: "add_workflow_action",
						Status: model.NodeDefault,
					},
					{
						Id:     "out1",
						Bot:    "dev",
						RuleId: "out_workflow_action",
						Status: model.NodeDefault,
					},
					{
						Id:     "out2",
						Bot:    "dev",
						RuleId: "out_workflow_action",
						Status: model.NodeDefault,
					},
				},
				Edges: []*model.Edge{
					{
						Id:     "edge-1",
						Source: "input",
						Target: "add_two_number",
					},
					{
						Id:     "edge-2",
						Source: "add_two_number",
						Target: "out1",
					},
					{
						Id:     "edge-3",
						Source: "add_two_number",
						Target: "out2",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "fail",
			args: args{
				code: failYaml,
			},
			want:    nil,
			want1:   nil,
			want2:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := ParseYamlWorkflow(tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseYamlWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseYamlWorkflow() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ParseYamlWorkflow() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("ParseYamlWorkflow() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestMetaDataValidate(t *testing.T) {
	okMeta, err := parseWorkflowMetadata(okYaml)
	if err != nil {
		t.Fatal(err)
	}

	failMeta, err := parseWorkflowMetadata(failYaml)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		meta types.WorkflowMetadata
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				meta: okMeta,
			},
			wantErr: false,
		},
		{
			name: "fail",
			args: args{
				meta: failMeta,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MetaDataValidate(tt.args.meta); (err != nil) != tt.wantErr {
				t.Errorf("MetaDataValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_pipelineFormatValidate(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "six nodes",
			args: args{
				s: "input -> add_two_number -> out1 -> debug -> archive -> end",
			},
			want: true,
		},
		{
			name: "five nodes",
			args: args{
				s: "input -> add_two_number -> out1 -> debug -> archive",
			},
			want: true,
		},
		{
			name: "four nodes",
			args: args{
				s: "input -> add_two_number -> out1 -> debug",
			},
			want: true,
		},
		{
			name: "three nodes",
			args: args{
				s: "input -> add_two_number -> out1",
			},
			want: true,
		},
		{
			name: "two nodes",
			args: args{
				s: "add_two_number -> out1",
			},
			want: true,
		},
		{
			name: "one node",
			args: args{
				s: "add_two_number",
			},
			want: false,
		},
		{
			name: "spaces",
			args: args{
				s: "   add_two_number  ->    out1 ",
			},
			want: true,
		},
		{
			name: "error first",
			args: args{
				s: "-> add_two_number -> out1",
			},
			want: false,
		},
		{
			name: "error last",
			args: args{
				s: "add_two_number -> out1  ->",
			},
			want: false,
		},
		{
			name: "empty node",
			args: args{
				s: "->",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineFormatValidate(tt.args.s); got != tt.want {
				t.Errorf("pipelineFormatValidate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parsePipelineEdges(t *testing.T) {
	type args struct {
		pipeline string
	}
	tests := []struct {
		name string
		args args
		want [][2]string
	}{
		{
			name: "case1",
			args: args{
				pipeline: "a -> b -> c -> d",
			},
			want: [][2]string{
				{"a", "b"},
				{"b", "c"},
				{"c", "d"},
			},
		},
		{
			name: "case2",
			args: args{
				pipeline: "a -> b -> c",
			},
			want: [][2]string{
				{"a", "b"},
				{"b", "c"},
			},
		},
		{
			name: "case3",
			args: args{
				pipeline: "a -> b",
			},
			want: [][2]string{
				{"a", "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePipelineEdges(tt.args.pipeline); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePipelineEdges() = %v, want %v", got, tt.want)
			}
		})
	}
}
