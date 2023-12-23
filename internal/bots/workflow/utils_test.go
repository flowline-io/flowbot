package workflow

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"reflect"
	"testing"
)

func TestParseYamlWorkflow(t *testing.T) {
	type args struct {
		code string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.Workflow
		want1   *model.WorkflowTrigger
		want2   *model.Dag
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				code: `---
name: example
describe: do something...

trigger:
  type: cron # cron, manual, webhook
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
    action: out_workflow_action@dev`,
			},
			want: &model.Workflow{
				Name:     "example",
				Describe: "do something...",
			},
			want1: &model.WorkflowTrigger{
				Type: model.TriggerCron,
				Rule: model.JSON{
					"spec": "* * * * *",
				},
				State: model.WorkflowTriggerEnable,
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
					},
					{
						Id:     "add_two_number",
						Bot:    "dev",
						RuleId: "add_workflow_action",
					},
					{
						Id:     "out1",
						Bot:    "dev",
						RuleId: "out_workflow_action",
					},
					{
						Id:     "out2",
						Bot:    "dev",
						RuleId: "out_workflow_action",
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
